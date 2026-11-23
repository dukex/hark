package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
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

	me        string
	started   time.Time
	lastEvent int64
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
	} `json:"payload"`
}

var kinds = map[string]string{"Issue": "issue", "PullRequest": "pr"}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	return append(w.notifications(ctx), w.ownEvents(ctx)...)
}

func (w *Watcher) notifications(ctx context.Context) []hark.Event {
	var threads []notification
	q := url.Values{"since": {w.started.Add(-backlog).Format(time.RFC3339)}}
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
	if err := w.do(ctx, http.MethodGet, w.API+"/users/"+w.me+"/events", &list); err != nil {
		return []hark.Event{w.failure(err)}
	}
	var events []hark.Event
	for i := len(list) - 1; i >= 0; i-- {
		o := list[i]
		id, _ := strconv.ParseInt(o.ID, 10, 64)
		if id <= w.lastEvent || o.CreatedAt.Before(w.started) {
			continue
		}
		w.lastEvent = id
		p := o.Payload
		kind, target := "", p.PullRequest
		switch {
		case o.Type == "PullRequestEvent" && p.Action == "opened":
			kind = "pr.opened"
		case o.Type == "PullRequestEvent" && p.Action == "closed":
			kind = "pr.closed"
		case o.Type == "IssuesEvent" && p.Action == "closed":
			kind, target = "issue.closed", p.Issue
		default:
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
			Number: s.Number, Actor: w.me, Text: s.Title, URL: s.HTMLURL,
		})
	}
	return events
}

func (w *Watcher) thread(ctx context.Context, n notification) []hark.Event {
	var events []hark.Event
	if kind, ok := kinds[n.Subject.Type]; ok {
		var err error
		if events, err = w.discussion(ctx, n, kind); err != nil {
			return []hark.Event{w.failure(err)}
		}
	} else if n.Subject.Type == "CheckSuite" || n.Subject.Type == "WorkflowRun" {
		events = []hark.Event{ci(n)}
	} else {
		return nil
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
	var s subject
	if err := w.do(ctx, http.MethodGet, n.Subject.URL, &s); err != nil {
		return nil, err
	}
	base := hark.Event{Provider: "github", Repo: n.Repository.FullName, Number: s.Number, Watching: s.User.Login != w.me}
	var events []hark.Event
	if strings.Contains(n.Subject.LatestCommentURL, "/comments/") {
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
	if len(events) == 0 {
		e := base
		e.Time, e.Kind, e.Text, e.URL = n.UpdatedAt, kind+".updated", n.Reason, s.HTMLURL
		events = append(events, e)
	}
	return events, nil
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
