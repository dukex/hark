package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
)

var unescape = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&")

type Watcher struct {
	API      string
	Token    string
	State    string
	Interval time.Duration
	Now      func() time.Time
	Since    time.Time
	Filter   string

	me    string
	state state
}

type state struct {
	Cursor time.Time `json:"cursor"`
	Seen   []string  `json:"seen"`
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

type identity struct {
	UserID string `json:"user_id"`
	User   string `json:"user"`
	Team   string `json:"team"`
}

type match struct {
	Type    string `json:"type"`
	Channel struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		IsIM   bool   `json:"is_im"`
		IsMPIM bool   `json:"is_mpim"`
	} `json:"channel"`
	User      string `json:"user"`
	Username  string `json:"username"`
	TS        string `json:"ts"`
	Text      string `json:"text"`
	Permalink string `json:"permalink"`
}

type results struct {
	Messages struct {
		Total   int     `json:"total"`
		Matches []match `json:"matches"`
		Paging  struct {
			Pages int `json:"pages"`
		} `json:"paging"`
	} `json:"messages"`
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	var me identity
	if err := w.call(ctx, "auth.test", nil, &me); err != nil {
		return err
	}
	w.me = me.UserID
	if w.Filter != "" {
		var probe results
		if err := w.call(ctx, "search.messages", url.Values{"query": {w.mention() + " " + w.Filter}, "count": {"1"}}, &probe); err != nil {
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

func (w *Watcher) mention() string { return "<@" + w.me + ">" }

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	next := w.Now().Add(-lag)
	if next.Before(w.state.Cursor) {
		next = w.state.Cursor
	}
	after := " after:" + w.state.Cursor.UTC().AddDate(0, 0, -2).Format(time.DateOnly)
	found := map[string]bool{}
	var events []hark.Event
	var seen []string
	for _, base := range []string{w.mention(), "to:me"} {
		q := url.Values{"query": {strings.TrimSpace(base + after + " " + w.Filter)}, "sort": {"timestamp"}, "sort_dir": {"desc"}, "count": {"100"}}
		for page, older := 1, false; !older; page++ {
			q.Set("page", strconv.Itoa(page))
			var res results
			if err := w.call(ctx, "search.messages", q, &res); err != nil {
				return []hark.Event{w.failure(err)}
			}
			for _, m := range res.Messages.Matches {
				t, err := parseTS(m.TS)
				if err != nil {
					return []hark.Event{w.failure(err)}
				}
				key := m.Channel.ID + ":" + m.TS
				if older = t.Before(w.state.Cursor); older || found[key] {
					continue
				}
				found[key] = true
				if !t.Before(next) {
					seen = append(seen, key)
				}
				if e, ok := w.event(m, t, base == w.mention()); ok && !slices.Contains(w.state.Seen, key) {
					events = append(events, e)
				}
			}
			older = older || page >= res.Messages.Paging.Pages
		}
	}
	slices.SortStableFunc(events, func(a, b hark.Event) int { return a.Time.Compare(b.Time) })
	w.state.Cursor, w.state.Seen = next, seen
	return events
}

func (w *Watcher) event(m match, t time.Time, mentioned bool) (hark.Event, bool) {
	direct := m.Type == "im" || m.Channel.IsIM || m.Channel.IsMPIM
	e := hark.Event{Time: t.UTC(), Provider: "slack", Actor: m.Username, Text: unescape.Replace(m.Text), URL: m.Permalink}
	if !direct {
		e.Repo = "#" + m.Channel.Name
	}
	switch {
	case m.User == w.me:
		return e, false
	case mentioned || strings.Contains(m.Text, w.mention()) || strings.Contains(m.Text, "<@"+w.me+"|"):
		e.Kind = "slack.mentioned"
	case direct:
		e.Kind = "slack.dm"
	default:
		return e, false
	}
	return e, true
}

func parseTS(ts string) (time.Time, error) {
	sec, micro, _ := strings.Cut(ts, ".")
	s, err := strconv.ParseInt(sec, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("message ts %q: %w", ts, err)
	}
	us, _ := strconv.ParseInt(micro, 10, 64)
	return time.Unix(s, us*1000), nil
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "slack", Kind: "error", Text: err.Error()}
}

func (w *Watcher) call(ctx context.Context, method string, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.API+"/"+method+"?"+q.Encode(), nil)
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
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var status struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}
	if err := json.Unmarshal(body, &status); err != nil {
		return err
	}
	if !status.OK {
		return errors.New(method + ": " + status.Error)
	}
	return json.Unmarshal(body, v)
}

func (w *Watcher) Doctor(ctx context.Context) []hark.Check {
	var me identity
	if err := w.call(ctx, "auth.test", nil, &me); err != nil {
		return []hark.Check{{Name: "token", Err: err}}
	}
	w.me = me.UserID
	after := " after:" + w.Now().UTC().AddDate(0, 0, -8).Format(time.DateOnly)
	var mentions, dms results
	err := w.call(ctx, "search.messages", url.Values{"query": {w.mention() + after}, "count": {"1"}}, &mentions)
	if err == nil {
		err = w.call(ctx, "search.messages", url.Values{"query": {"to:me" + after}, "count": {"1"}}, &dms)
	}
	return []hark.Check{
		{Name: "token", Detail: "logged in as " + me.User + " on " + me.Team},
		{Name: "search", Detail: fmt.Sprintf("%d mentions and %d DMs in the last 7 days", mentions.Messages.Total, dms.Messages.Total), Err: err},
		hark.CheckState(w.State),
	}
}
