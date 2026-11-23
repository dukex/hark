package todoist

import (
	"cmp"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/dukex/hark"
)

const (
	backlog = time.Hour
	taskURL = "https://app.todoist.com/app/task/"
	watched = `["item:added","item:completed","item:updated","note:added"]`
)

type Watcher struct {
	API      string
	Token    string
	State    string
	Interval time.Duration
	Now      func() time.Time

	state state
}

type state struct {
	Cursor time.Time `json:"cursor"`
	Seen   []string  `json:"seen"`
	DueDay string    `json:"due_day"`
}

func New(api, token, statePath string) *Watcher {
	return &Watcher{
		API:      strings.TrimSuffix(api, "/"),
		Token:    token,
		State:    statePath,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	if err := hark.LoadState(w.State, &w.state); err != nil {
		return err
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

type activity struct {
	ObjectType   string    `json:"object_type"`
	EventType    string    `json:"event_type"`
	ObjectID     string    `json:"object_id"`
	ParentItemID string    `json:"parent_item_id"`
	EventDate    time.Time `json:"event_date"`
	ExtraData    struct {
		Content     string `json:"content"`
		DueDate     string `json:"due_date"`
		LastDueDate string `json:"last_due_date"`
	} `json:"extra_data"`
}

type task struct {
	ID      string `json:"id"`
	Content string `json:"content"`
}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	return append(w.activity(ctx), w.dueToday(ctx)...)
}

func (w *Watcher) activity(ctx context.Context) []hark.Event {
	q := url.Values{
		"object_event_types": {watched},
		"date_from":          {w.state.Cursor.UTC().Format(time.RFC3339Nano)},
		"annotate_notes":     {"true"},
		"limit":              {"100"},
	}
	var all []activity
	for {
		var page struct {
			Results    []activity `json:"results"`
			NextCursor string     `json:"next_cursor"`
		}
		if err := w.get(ctx, "/activities?"+q.Encode(), &page); err != nil {
			return []hark.Event{w.failure(err)}
		}
		all = append(all, page.Results...)
		if page.NextCursor == "" {
			break
		}
		q.Set("cursor", page.NextCursor)
	}
	slices.SortStableFunc(all, func(a, b activity) int { return a.EventDate.Compare(b.EventDate) })

	var events []hark.Event
	for _, a := range all {
		key := strings.Join([]string{a.ObjectType, a.EventType, a.ObjectID, a.EventDate.Format(time.RFC3339Nano)}, ":")
		if a.EventDate.Before(w.state.Cursor) || slices.Contains(w.state.Seen, key) {
			continue
		}
		if a.EventDate.After(w.state.Cursor) {
			w.state.Cursor, w.state.Seen = a.EventDate, nil
		}
		w.state.Seen = append(w.state.Seen, key)
		if e, ok := w.event(a); ok {
			events = append(events, e)
		}
	}
	return events
}

func (w *Watcher) event(a activity) (hark.Event, bool) {
	e := hark.Event{Time: a.EventDate, Provider: "todoist", Text: a.ExtraData.Content, URL: taskURL + a.ObjectID}
	switch a.ObjectType + ":" + a.EventType {
	case "item:added":
		e.Kind = "task.created"
	case "item:completed":
		e.Kind = "task.completed"
	case "item:updated":
		if a.ExtraData.DueDate == a.ExtraData.LastDueDate {
			return e, false
		}
		e.Kind = "task.rescheduled"
		e.Text = fmt.Sprintf("%s, due %s", a.ExtraData.Content, cmp.Or(w.localDate(a.ExtraData.DueDate), "removed"))
	default:
		e.Kind, e.URL = "task.commented", taskURL+a.ParentItemID
	}
	return e, true
}

func (w *Watcher) localDate(due string) string {
	t, err := time.Parse(time.RFC3339Nano, due)
	if err != nil {
		return due
	}
	return t.In(w.Now().Location()).Format(time.DateOnly)
}

func (w *Watcher) dueToday(ctx context.Context) []hark.Event {
	today := w.Now().Format(time.DateOnly)
	if w.state.DueDay == today {
		return nil
	}
	var page struct {
		Results []task `json:"results"`
	}
	if err := w.get(ctx, "/tasks/filter?"+url.Values{"query": {"today"}, "limit": {"200"}}.Encode(), &page); err != nil {
		return []hark.Event{w.failure(err)}
	}
	var events []hark.Event
	for _, t := range page.Results {
		events = append(events, hark.Event{Time: w.Now(), Provider: "todoist", Kind: "task.due", Text: t.Content, URL: taskURL + t.ID})
	}
	w.state.DueDay = today
	return events
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "todoist", Kind: "error", Text: err.Error()}
}

func (w *Watcher) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.API+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+w.Token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", req.URL.Path, res.Status)
	}
	return json.NewDecoder(res.Body).Decode(v)
}

func (w *Watcher) Doctor(ctx context.Context) []hark.Check {
	var tasks struct {
		Results []task `json:"results"`
	}
	if err := w.get(ctx, "/tasks/filter?query=today&limit=1", &tasks); err != nil {
		return []hark.Check{{Name: "token", Err: err}}
	}
	var activity struct {
		Results []activity `json:"results"`
	}
	return []hark.Check{
		{Name: "token", Detail: "valid"},
		{Name: "activity log", Detail: "readable", Err: w.get(ctx, "/activities?limit=1", &activity)},
		hark.CheckState(w.State),
	}
}
