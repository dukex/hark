package github

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/dukex/hark"
)

const backlog = time.Hour

type Watcher struct {
	API      string
	Token    string
	Client   *http.Client
	Interval time.Duration
	Now      func() time.Time
	Since    time.Time
	Filter   Filter

	me         string
	started    time.Time
	seenEvents map[string]bool
}

func New(api, token string) *Watcher {
	return &Watcher{
		API:      strings.TrimSuffix(api, "/"),
		Token:    token,
		Client:   http.DefaultClient,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

type user struct {
	Login string `json:"login"`
}

type notification struct {
	ID         string    `json:"id"`
	Reason     string    `json:"reason"`
	UpdatedAt  time.Time `json:"updated_at"`
	LastReadAt time.Time `json:"last_read_at"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Subject struct {
		Type             string `json:"type"`
		Title            string `json:"title"`
		URL              string `json:"url"`
		LatestCommentURL string `json:"latest_comment_url"`
	} `json:"subject"`
}

type subject struct {
	Number   int       `json:"number"`
	Title    string    `json:"title"`
	User     user      `json:"user"`
	HTMLURL  string    `json:"html_url"`
	ClosedAt time.Time `json:"closed_at"`
	ClosedBy user      `json:"closed_by"`
	Merged   bool      `json:"merged"`
	MergedBy user      `json:"merged_by"`
}

type comment struct {
	User      user      `json:"user"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	HTMLURL   string    `json:"html_url"`
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	var me user
	if err := w.do(ctx, http.MethodGet, w.API+"/user", &me); err != nil {
		return err
	}
	w.me, w.started = me.Login, w.Now()
	for {
		for _, e := range w.poll(ctx) {
			emit(e)
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.Interval):
		}
	}
}

type ref struct {
	URL string `json:"url"`
}

type ownEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"created_at"`
	Repo      struct {
		Name string `json:"name"`
	} `json:"repo"`
	Payload struct {
		Action      string `json:"action"`
		PullRequest ref    `json:"pull_request"`
		Issue       ref    `json:"issue"`
		Review      struct {
			State   string `json:"state"`
			HTMLURL string `json:"html_url"`
		} `json:"review"`
	} `json:"payload"`
}

var kinds = map[string]string{"Issue": "issue", "PullRequest": "pr", "CheckSuite": "ci", "WorkflowRun": "ci"}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	return append(w.notifications(ctx), w.ownEvents(ctx)...)
}

func (w *Watcher) notifications(ctx context.Context) []hark.Event {
	var threads []notification
	q := url.Values{"since": {w.since(backlog).UTC().Format(time.RFC3339)}}
	if err := w.do(ctx, http.MethodGet, w.API+"/notifications?"+q.Encode(), &threads); err != nil {
		return []hark.Event{w.failure(err)}
	}
	var events []hark.Event
	for i := len(threads) - 1; i >= 0; i-- {
		events = append(events, w.thread(ctx, threads[i])...)
	}
	return events
}

func (w *Watcher) ownEvents(ctx context.Context) []hark.Event {
	var list []ownEvent
	if err := w.do(ctx, http.MethodGet, w.API+"/users/"+w.me+"/events?per_page=100", &list); err != nil {
		return []hark.Event{w.failure(err)}
	}
	var events []hark.Event
	seen := map[string]bool{}
	for i := len(list) - 1; i >= 0; i-- {
		o := list[i]
		seen[o.ID] = true
		if w.seenEvents[o.ID] || o.CreatedAt.Before(w.since(0)) {
			continue
		}
		p := o.Payload
		kind, target, link := "", p.PullRequest, ""
		switch {
		case o.Type == "PullRequestEvent" && p.Action == "opened":
			kind = "pr.opened"
		case o.Type == "PullRequestEvent" && p.Action == "closed":
			kind = "pr.closed"
		case o.Type == "PullRequestEvent" && p.Action == "merged":
			kind = "pr.merged"
		case o.Type == "PullRequestReviewEvent" && p.Action == "created" && reviewKinds[strings.ToUpper(p.Review.State)] != "":
			kind, link = reviewKinds[strings.ToUpper(p.Review.State)], p.Review.HTMLURL
		case o.Type == "IssuesEvent" && p.Action == "closed":
			kind, target = "issue.closed", p.Issue
		default:
			continue
		}
		if is, _, _ := strings.Cut(kind, "."); !w.Filter.match(o.Repo.Name, "", is) {
			continue
		}
		var s subject
		if err := w.do(ctx, http.MethodGet, target.URL, &s); err != nil {
			events = append(events, w.failure(err))
			continue
		}
		if kind == "pr.closed" && s.Merged {
			kind = "pr.merged"
		}
		events = append(events, hark.Event{
			Time: o.CreatedAt, Provider: "github", Kind: kind, Repo: o.Repo.Name,
			Number: s.Number, Actor: w.me, Text: s.Title, URL: cmp.Or(link, s.HTMLURL),
		})
	}
	w.seenEvents = seen
	return events
}

func (w *Watcher) thread(ctx context.Context, n notification) []hark.Event {
	kind, ok := kinds[n.Subject.Type]
	if !ok || !w.Filter.match(n.Repository.FullName, n.Reason, kind) {
		return nil
	}
	var events []hark.Event
	if kind == "ci" {
		events = []hark.Event{ci(n)}
	} else {
		var err error
		if events, err = w.discussion(ctx, n, kind); err != nil {
			return []hark.Event{w.failure(err)}
		}
	}
	if err := w.do(ctx, http.MethodPatch, w.API+"/notifications/threads/"+n.ID, nil); err != nil {
		events = append(events, w.failure(err))
	}
	return events
}

var ciResult = regexp.MustCompile(` workflow run(?:, Attempt #\d+)? (\w+) for `)

func ci(n notification) hark.Event {
	e := hark.Event{Time: n.UpdatedAt, Provider: "github", Kind: "ci." + n.Reason, Repo: n.Repository.FullName, Text: n.Subject.Title}
	if m := ciResult.FindStringSubmatch(n.Subject.Title); m != nil {
		e.Kind = "ci." + m[1]
	}
	return e
}

func (w *Watcher) discussion(ctx context.Context, n notification, kind string) ([]hark.Event, error) {
	prev := n.LastReadAt
	if floor := w.since(backlog); prev.Before(floor) {
		prev = floor
	}
	var s subject
	if err := w.do(ctx, http.MethodGet, n.Subject.URL, &s); err != nil {
		return nil, err
	}
	base := hark.Event{Provider: "github", Repo: n.Repository.FullName, Number: s.Number, Watching: s.User.Login != w.me}
	var events []hark.Event
	if strings.Contains(n.Subject.LatestCommentURL, "/issues/comments/") {
		var c comment
		if err := w.do(ctx, http.MethodGet, n.Subject.LatestCommentURL, &c); err != nil {
			return nil, err
		}
		if c.CreatedAt.After(prev) {
			e := base
			e.Time, e.Kind, e.Actor, e.Text, e.URL = c.CreatedAt, kind+".commented", c.User.Login, c.Body, c.HTMLURL
			events = append(events, e)
		}
	}
	if s.ClosedAt.After(prev) {
		e := base
		e.Time, e.Kind, e.Actor, e.URL = s.ClosedAt, kind+".closed", s.ClosedBy.Login, s.HTMLURL
		if s.Merged {
			e.Kind, e.Actor = kind+".merged", s.MergedBy.Login
		}
		events = append(events, e)
	}
	if kind == "pr" {
		reviews, err := w.reviews(ctx, base, n.Subject.URL, prev)
		if err != nil {
			return nil, err
		}
		events = append(events, reviews...)
	}
	if len(events) == 0 {
		e := base
		e.Time, e.Kind, e.Text, e.URL = n.UpdatedAt, kind+".updated", n.Reason, s.HTMLURL
		events = append(events, e)
	}
	slices.SortStableFunc(events, func(a, b hark.Event) int { return a.Time.Compare(b.Time) })
	return events, nil
}

type review struct {
	ID          int64     `json:"id"`
	User        user      `json:"user"`
	State       string    `json:"state"`
	Body        string    `json:"body"`
	SubmittedAt time.Time `json:"submitted_at"`
	HTMLURL     string    `json:"html_url"`
}

type reviewComment struct {
	ReviewID int64  `json:"pull_request_review_id"`
	Body     string `json:"body"`
}

var (
	reviewKinds = map[string]string{"APPROVED": "pr.approved", "CHANGES_REQUESTED": "pr.changes_requested"}
	htmlComment = regexp.MustCompile(`(?s)<!--.*?-->`)
)

func (w *Watcher) reviews(ctx context.Context, base hark.Event, pull string, prev time.Time) ([]hark.Event, error) {
	var reviews []review
	if err := w.do(ctx, http.MethodGet, pull+"/reviews?per_page=100", &reviews); err != nil {
		return nil, err
	}
	var comments []reviewComment
	q := url.Values{"since": {prev.UTC().Format(time.RFC3339)}, "per_page": {"100"}}
	if err := w.do(ctx, http.MethodGet, pull+"/comments?"+q.Encode(), &comments); err != nil {
		return nil, err
	}
	inline := map[int64][]string{}
	for _, c := range comments {
		inline[c.ReviewID] = append(inline[c.ReviewID], c.Body)
	}
	var events []hark.Event
	for _, r := range reviews {
		if !r.SubmittedAt.After(prev) || r.User.Login == w.me {
			continue
		}
		e := base
		e.Time, e.Kind, e.Actor, e.URL = r.SubmittedAt, cmp.Or(reviewKinds[r.State], "pr.reviewed"), r.User.Login, r.HTMLURL
		e.Text = strings.TrimSpace(htmlComment.ReplaceAllString(r.Body, ""))
		if c := inline[r.ID]; len(c) > 0 {
			e.Text = fmt.Sprintf("%s (%d inline comments)", cmp.Or(e.Text, c[0]), len(c))
		}
		events = append(events, e)
	}
	return events, nil
}

func (w *Watcher) since(backlog time.Duration) time.Time {
	if !w.Since.IsZero() {
		return w.Since
	}
	return w.started.Add(-backlog)
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "github", Kind: "error", Text: err.Error()}
}

func (w *Watcher) do(ctx context.Context, method, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	res, err := w.Client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode/100 != 2 {
		return fmt.Errorf("%s %s: %s", method, req.URL.Path, res.Status)
	}
	if v == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(v)
}

func (w *Watcher) Doctor(ctx context.Context) []hark.Check {
	var me user
	if err := w.do(ctx, http.MethodGet, w.API+"/user", &me); err != nil {
		return []hark.Check{{Name: "token", Err: err}}
	}
	var threads []notification
	return []hark.Check{
		{Name: "token", Detail: "logged in as " + me.Login},
		{Name: "notifications", Detail: "readable", Err: w.do(ctx, http.MethodGet, w.API+"/notifications?per_page=1", &threads)},
	}
}
