package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dukex/hark"
)

const (
	backlog = time.Hour
	lag     = 5 * time.Minute
	alerts  = "source:alert"
)

type Watcher struct {
	API      string
	App      string
	APIKey   string
	AppKey   string
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

func New(api, app, apiKey, appKey, statePath string) *Watcher {
	return &Watcher{
		API:      strings.TrimSuffix(api, "/"),
		App:      strings.TrimSuffix(app, "/"),
		APIKey:   apiKey,
		AppKey:   appKey,
		State:    statePath,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	if w.Filter != "" {
		var probe struct{}
		if err := w.get(ctx, "/api/v2/events?"+url.Values{"filter[query]": {w.query()}, "page[limit]": {"1"}}.Encode(), &probe); err != nil {
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

type event struct {
	ID         string `json:"id"`
	Attributes struct {
		Timestamp  time.Time `json:"timestamp"`
		Attributes struct {
			Title   string `json:"title"`
			Monitor struct {
				ID         int64 `json:"id"`
				Transition struct {
					From string `json:"source_state"`
					To   string `json:"destination_state"`
				} `json:"transition"`
			} `json:"monitor"`
		} `json:"attributes"`
	} `json:"attributes"`
}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	now := w.Now()
	next := now.Add(-lag)
	if next.Before(w.state.Cursor) {
		next = w.state.Cursor
	}
	q := url.Values{
		"filter[query]": {w.query()},
		"filter[from]":  {strconv.FormatInt(w.state.Cursor.UnixMilli(), 10)},
		"filter[to]":    {strconv.FormatInt(now.UnixMilli(), 10)},
		"sort":          {"timestamp"},
		"page[limit]":   {"1000"},
	}
	var events []hark.Event
	var seen []string
	for {
		var page struct {
			Data []event `json:"data"`
			Meta struct {
				Page struct {
					After string `json:"after"`
				} `json:"page"`
			} `json:"meta"`
		}
		if err := w.get(ctx, "/api/v2/events?"+q.Encode(), &page); err != nil {
			return []hark.Event{w.failure(err)}
		}
		for _, e := range page.Data {
			a := e.Attributes.Attributes
			if !e.Attributes.Timestamp.Before(next) {
				seen = append(seen, e.ID)
			}
			if e.Attributes.Timestamp.Before(w.state.Cursor) || slices.Contains(w.state.Seen, e.ID) || a.Monitor.Transition.From == a.Monitor.Transition.To {
				continue
			}
			kind := strings.ReplaceAll(strings.ToLower(a.Monitor.Transition.To), " ", "_")
			if kind == "ok" {
				kind = "recovered"
			}
			events = append(events, hark.Event{
				Time:     e.Attributes.Timestamp.UTC(),
				Provider: "datadog",
				Kind:     "datadog." + kind,
				Text:     a.Title,
				URL:      fmt.Sprintf("%s/monitors/%d", w.App, a.Monitor.ID),
			})
		}
		if page.Meta.Page.After == "" {
			break
		}
		q.Set("page[cursor]", page.Meta.Page.After)
	}
	w.state.Cursor, w.state.Seen = next, seen
	return events
}

func (w *Watcher) query() string {
	if w.Filter == "" {
		return alerts
	}
	return alerts + " (" + w.Filter + ")"
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "datadog", Kind: "error", Text: err.Error()}
}

func (w *Watcher) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.API+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("DD-API-KEY", w.APIKey)
	req.Header.Set("DD-APPLICATION-KEY", w.AppKey)
	req.Header.Set("Accept", "application/json")
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
	var valid struct{}
	if err := w.get(ctx, "/api/v1/validate", &valid); err != nil {
		return []hark.Check{{Name: "api key", Err: err}}
	}
	var events struct{}
	return []hark.Check{
		{Name: "api key", Detail: "valid"},
		{Name: "events", Detail: "monitor events are readable", Err: w.get(ctx, "/api/v2/events?"+url.Values{"filter[query]": {alerts}, "page[limit]": {"1"}}.Encode(), &events)},
		hark.CheckState(w.State),
	}
}
