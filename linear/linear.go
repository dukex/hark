package linear

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/dukex/hark"
)

const (
	backlog = time.Hour
	lag     = 5 * time.Minute
)

const viewerQuery = `query { viewer { name } }`

const notificationsQuery = `query($filter: NotificationFilter, $after: String, $first: Int) {
  notifications(filter: $filter, first: $first, after: $after, includeArchived: true) {
    nodes {
      id type createdAt
      actor { displayName }
      ... on IssueNotification {
        issue { identifier title url state { name } }
        comment { body url }
      }
    }
    pageInfo { hasNextPage endCursor }
  }
}`

var kinds = map[string]string{
	"issueMention":        "linear.mentioned",
	"issueCommentMention": "linear.mentioned",
	"issueAssignedToYou":  "linear.assigned",
	"issueNewComment":     "linear.commented",
	"issueStatusChanged":  "linear.transitioned",
}

type Watcher struct {
	API      string
	Key      string
	State    string
	Interval time.Duration
	Now      func() time.Time
	Since    time.Time
	Filter   string

	state state
}

type state struct {
	Cursor time.Time `json:"cursor"`
	Seen   []string  `json:"seen"`
}

func New(api, key, statePath string) *Watcher {
	return &Watcher{
		API:      api,
		Key:      key,
		State:    statePath,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

type notification struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	Actor     *struct {
		DisplayName string `json:"displayName"`
	} `json:"actor"`
	Issue *struct {
		Identifier string `json:"identifier"`
		Title      string `json:"title"`
		URL        string `json:"url"`
		State      struct {
			Name string `json:"name"`
		} `json:"state"`
	} `json:"issue"`
	Comment *struct {
		Body string `json:"body"`
		URL  string `json:"url"`
	} `json:"comment"`
}

type notifications struct {
	Notifications struct {
		Nodes    []notification `json:"nodes"`
		PageInfo struct {
			HasNextPage bool   `json:"hasNextPage"`
			EndCursor   string `json:"endCursor"`
		} `json:"pageInfo"`
	} `json:"notifications"`
}

type viewer struct {
	Viewer struct {
		Name string `json:"name"`
	} `json:"viewer"`
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	var me viewer
	if err := w.call(ctx, viewerQuery, nil, &me); err != nil {
		return err
	}
	if w.Filter != "" {
		var filter map[string]any
		if json.Unmarshal([]byte(w.Filter), &filter) != nil || filter == nil {
			return fmt.Errorf("--filter %q: not a JSON object", w.Filter)
		}
		var probe notifications
		if err := w.call(ctx, notificationsQuery, map[string]any{"filter": filter, "first": 1}, &probe); err != nil {
			return fmt.Errorf("--filter %q: %w", w.Filter, err)
		}
	}
	if err := hark.LoadState(w.State, &w.state); err != nil {
		return err
	}
	if !w.Since.IsZero() {
		w.state.Cursor, w.state.Seen = w.Since, nil
	}
	if w.state.Cursor.IsZero() {
		w.state.Cursor = w.Now().Add(-backlog)
	}
	for {
		for _, e := range w.poll(ctx) {
			emit(e)
		}
		if err := hark.SaveState(w.State, w.state); err != nil {
			emit(w.failure(err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.Interval):
		}
	}
}

func (w *Watcher) where(from time.Time) map[string]any {
	since := map[string]any{"createdAt": map[string]any{"gte": from.UTC()}}
	var filter map[string]any
	if json.Unmarshal([]byte(w.Filter), &filter) != nil || filter == nil {
		return since
	}
	return map[string]any{"and": []any{since, filter}}
}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	next := w.Now().Add(-lag)
	if next.Before(w.state.Cursor) {
		next = w.state.Cursor
	}
	vars := map[string]any{"filter": w.where(w.state.Cursor), "first": 100}
	var events []hark.Event
	var seen []string
	for {
		var res notifications
		if err := w.call(ctx, notificationsQuery, vars, &res); err != nil {
			return []hark.Event{w.failure(err)}
		}
		for _, n := range res.Notifications.Nodes {
			if n.CreatedAt.Before(w.state.Cursor) {
				continue
			}
			if !n.CreatedAt.Before(next) {
				seen = append(seen, n.ID)
			}
			if !slices.Contains(w.state.Seen, n.ID) {
				events = append(events, event(n))
			}
		}
		if !res.Notifications.PageInfo.HasNextPage {
			break
		}
		vars["after"] = res.Notifications.PageInfo.EndCursor
	}
	slices.SortStableFunc(events, func(a, b hark.Event) int { return a.Time.Compare(b.Time) })
	w.state.Cursor, w.state.Seen = next, seen
	return events
}

func event(n notification) hark.Event {
	e := hark.Event{Time: n.CreatedAt.UTC(), Provider: "linear", Kind: kinds[n.Type]}
	if e.Kind == "" {
		e.Kind = "linear." + snake(strings.TrimPrefix(n.Type, "issue"))
	}
	if n.Actor != nil {
		e.Actor = strings.Join(strings.Fields(n.Actor.DisplayName), "")
	}
	if n.Issue != nil {
		e.Repo, e.Text, e.URL = n.Issue.Identifier, n.Issue.Title, n.Issue.URL
		if n.Type == "issueStatusChanged" {
			e.Text += ": " + n.Issue.State.Name
		}
	}
	if n.Comment != nil {
		e.Text, e.URL = n.Comment.Body, n.Comment.URL
	}
	return e
}

func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if unicode.IsUpper(r) && i > 0 {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return b.String()
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "linear", Kind: "error", Text: err.Error()}
}

func (w *Watcher) call(ctx context.Context, query string, vars map[string]any, v any) error {
	body, _ := json.Marshal(map[string]any{"query": query, "variables": vars})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.API, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", w.Key)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var out struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(data, &out) == nil && len(out.Errors) > 0 {
		return errors.New(out.Errors[0].Message)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("POST %s: %s", w.API, res.Status)
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return err
	}
	return json.Unmarshal(out.Data, v)
}

func (w *Watcher) Doctor(ctx context.Context) []hark.Check {
	var me viewer
	if err := w.call(ctx, viewerQuery, nil, &me); err != nil {
		return []hark.Check{{Name: "api key", Err: err}}
	}
	var probe notifications
	err := w.call(ctx, notificationsQuery, map[string]any{"filter": w.where(w.Now().Add(-7 * 24 * time.Hour)), "first": 1}, &probe)
	return []hark.Check{
		{Name: "api key", Detail: "logged in as " + me.Viewer.Name},
		{Name: "notifications", Detail: "readable", Err: err},
		hark.CheckState(w.State),
	}
}
