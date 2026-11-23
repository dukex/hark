package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dukex/hark"
)

const overlap = time.Minute

type Watcher struct {
	API      string
	Token    string
	Client   *http.Client
	Interval time.Duration
	Now      func() time.Time

	me        string
	started   time.Time
	since     time.Time
	seen      map[string]time.Time
	lastEvent int64
}

func New(api, token string) *Watcher {
	return &Watcher{
		API:      strings.TrimSuffix(api, "/"),
		Token:    token,
		Client:   http.DefaultClient,
		Interval: time.Minute,
		Now:      time.Now,
		seen:     map[string]time.Time{},
	}
}

type user struct {
	Login string `json:"login"`
}

type notification struct {
	ID         string    `json:"id"`
	Reason     string    `json:"reason"`
	UpdatedAt  time.Time `json:"updated_at"`
	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
	Subject struct {
		Type             string `json:"type"`
		URL              string `json:"url"`
		LatestCommentURL string `json:"latest_comment_url"`
	} `json:"subject"`
}

type subject struct {
	Number   int       `json:"number"`
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
	if err := w.get(ctx, w.API+"/user", &me); err != nil {
		return err
	}
	w.me, w.started = me.Login, w.Now()
	w.since = w.started
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

type item struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	HTMLURL string `json:"html_url"`
	Merged  bool   `json:"merged"`
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
		PullRequest item   `json:"pull_request"`
		Issue       item   `json:"issue"`
	} `json:"payload"`
}

var kinds = map[string]string{"Issue": "issue", "PullRequest": "pr"}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	return append(w.notifications(ctx), w.ownEvents(ctx)...)
}

func (w *Watcher) notifications(ctx context.Context) []hark.Event {
	polledAt := w.Now()
	var threads []notification
	q := url.Values{"all": {"true"}, "since": {w.since.Add(-overlap).Format(time.RFC3339)}}
	if err := w.get(ctx, w.API+"/notifications?"+q.Encode(), &threads); err != nil {
		return []hark.Event{w.failure(err)}
	}
	var events []hark.Event
	for i := len(threads) - 1; i >= 0; i-- {
		events = append(events, w.thread(ctx, threads[i])...)
	}
	w.since = polledAt
	return events
}

func (w *Watcher) ownEvents(ctx context.Context) []hark.Event {
	var list []ownEvent
	if err := w.get(ctx, w.API+"/users/"+w.me+"/events", &list); err != nil {
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
		kind, it := "", p.PullRequest
		switch {
		case o.Type == "PullRequestEvent" && p.Action == "opened":
			kind = "pr.opened"
		case o.Type == "PullRequestEvent" && p.Action == "closed" && it.Merged:
			kind = "pr.merged"
		case o.Type == "PullRequestEvent" && p.Action == "closed":
			kind = "pr.closed"
		case o.Type == "IssuesEvent" && p.Action == "closed":
			kind, it = "issue.closed", p.Issue
		default:
			continue
		}
		events = append(events, hark.Event{
			Time: o.CreatedAt, Provider: "github", Kind: kind, Repo: o.Repo.Name,
			Number: it.Number, Actor: w.me, Text: it.Title, URL: it.HTMLURL,
		})
	}
	return events
}

func (w *Watcher) thread(ctx context.Context, n notification) []hark.Event {
	kind, ok := kinds[n.Subject.Type]
	prev, known := w.seen[n.ID]
	if !ok || (known && !n.UpdatedAt.After(prev)) {
		return nil
	}
	if !known {
		prev = w.since
	}
	var s subject
	if err := w.get(ctx, n.Subject.URL, &s); err != nil {
		return []hark.Event{w.failure(err)}
	}
	base := hark.Event{Provider: "github", Repo: n.Repository.FullName, Number: s.Number}
	var events []hark.Event
	if strings.Contains(n.Subject.LatestCommentURL, "/comments/") {
		var c comment
		if err := w.get(ctx, n.Subject.LatestCommentURL, &c); err != nil {
			return []hark.Event{w.failure(err)}
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
	w.seen[n.ID] = n.UpdatedAt
	return events
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "github", Kind: "error", Text: err.Error()}
}

func (w *Watcher) get(ctx context.Context, url string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
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
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", req.URL.Path, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(v)
}
