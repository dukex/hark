package jira

import (
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
	follows = "(assignee = currentUser() OR reporter = currentUser() OR watcher = currentUser())"
)

type Watcher struct {
	Site     string
	Email    string
	Token    string
	State    string
	Interval time.Duration
	Now      func() time.Time

	me    string
	state state
}

type state struct {
	Cursor time.Time `json:"cursor"`
	Seen   []string  `json:"seen"`
}

func New(site, email, token, statePath string) *Watcher {
	return &Watcher{
		Site:     strings.TrimSuffix(site, "/"),
		Email:    email,
		Token:    token,
		State:    statePath,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	var me user
	if err := w.get(ctx, "/rest/api/3/myself", &me); err != nil {
		return err
	}
	w.me = me.AccountID
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

type jiraTime struct{ time.Time }

func (t *jiraTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse("2006-01-02T15:04:05.000-0700", s)
	t.Time = parsed
	return err
}

type user struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
}

type comment struct {
	ID      string   `json:"id"`
	Author  user     `json:"author"`
	Body    adf      `json:"body"`
	Created jiraTime `json:"created"`
}

type issue struct {
	Key    string `json:"key"`
	Fields struct {
		Summary string `json:"summary"`
	} `json:"fields"`
	Changelog struct {
		Histories []struct {
			ID      string   `json:"id"`
			Author  user     `json:"author"`
			Created jiraTime `json:"created"`
			Items   []struct {
				Field      string `json:"field"`
				FromString string `json:"fromString"`
				To         string `json:"to"`
				ToString   string `json:"toString"`
			} `json:"items"`
		} `json:"histories"`
	} `json:"changelog"`
}

type adf struct {
	Type  string `json:"type"`
	Text  string `json:"text"`
	Attrs struct {
		ID   string `json:"id"`
		Text string `json:"text"`
	} `json:"attrs"`
	Content []adf `json:"content"`
}

func (n adf) plain(me string) (text string, mentioned bool) {
	var b strings.Builder
	var walk func(adf)
	walk = func(n adf) {
		switch n.Type {
		case "text":
			b.WriteString(n.Text)
		case "mention":
			b.WriteString(n.Attrs.Text)
			mentioned = mentioned || n.Attrs.ID == me
		}
		for _, c := range n.Content {
			walk(c)
		}
		b.WriteString(" ")
	}
	walk(n)
	return b.String(), mentioned
}

type found struct {
	key   string
	event hark.Event
}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	minutes := int(w.Now().Sub(w.state.Cursor).Minutes()) + 2
	q := url.Values{
		"jql":        {fmt.Sprintf("%s AND updated >= -%dm ORDER BY updated ASC", follows, minutes)},
		"fields":     {"summary"},
		"expand":     {"changelog"},
		"maxResults": {"50"},
	}
	var all []found
	for {
		var page struct {
			Issues        []issue `json:"issues"`
			NextPageToken string  `json:"nextPageToken"`
		}
		if err := w.get(ctx, "/rest/api/3/search/jql?"+q.Encode(), &page); err != nil {
			return []hark.Event{w.failure(err)}
		}
		for _, i := range page.Issues {
			var c struct {
				Comments []comment `json:"comments"`
			}
			if err := w.get(ctx, "/rest/api/3/issue/"+url.PathEscape(i.Key)+"/comment?orderBy=-created&maxResults=50", &c); err != nil {
				return []hark.Event{w.failure(err)}
			}
			all = append(all, w.found(i, c.Comments)...)
		}
		if page.NextPageToken == "" {
			break
		}
		q.Set("nextPageToken", page.NextPageToken)
	}
	slices.SortStableFunc(all, func(a, b found) int { return a.event.Time.Compare(b.event.Time) })

	var events []hark.Event
	for _, f := range all {
		if f.event.Time.Before(w.state.Cursor) || slices.Contains(w.state.Seen, f.key) {
			continue
		}
		if f.event.Time.After(w.state.Cursor) {
			w.state.Cursor, w.state.Seen = f.event.Time, nil
		}
		w.state.Seen = append(w.state.Seen, f.key)
		events = append(events, f.event)
	}
	return events
}

func (w *Watcher) found(i issue, comments []comment) []found {
	browse := w.Site + "/browse/" + i.Key
	base := hark.Event{Provider: "jira", Repo: i.Key, URL: browse}
	var all []found
	for _, c := range comments {
		if c.Author.AccountID == w.me {
			continue
		}
		text, mentioned := c.Body.plain(w.me)
		e := base
		e.Time, e.Kind, e.Text, e.URL = c.Created.UTC(), "jira.commented", c.Author.DisplayName+": "+text, browse+"?focusedCommentId="+c.ID
		if mentioned {
			e.Kind = "jira.mentioned"
		}
		all = append(all, found{i.Key + ":comment:" + c.ID, e})
	}
	for _, h := range i.Changelog.Histories {
		if h.Author.AccountID == w.me {
			continue
		}
		for _, item := range h.Items {
			e := base
			e.Time = h.Created.UTC()
			switch {
			case item.Field == "status":
				e.Kind = "jira.transitioned"
				e.Text = fmt.Sprintf("%s: %s to %s (%s)", i.Fields.Summary, item.FromString, item.ToString, h.Author.DisplayName)
			case item.Field == "assignee" && item.To == w.me:
				e.Kind = "jira.assigned"
				e.Text = fmt.Sprintf("%s (assigned by %s)", i.Fields.Summary, h.Author.DisplayName)
			default:
				continue
			}
			all = append(all, found{i.Key + ":history:" + h.ID + ":" + item.Field, e})
		}
	}
	return all
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "jira", Kind: "error", Text: err.Error()}
}

func (w *Watcher) get(ctx context.Context, path string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.Site+path, nil)
	if err != nil {
		return err
	}
	req.SetBasicAuth(w.Email, w.Token)
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
