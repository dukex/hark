package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dukex/hark"
)

var now = time.Date(2026, 11, 23, 21, 0, 0, 0, time.UTC)

type fakeJira struct {
	*httptest.Server
	mu      sync.Mutex
	pages   map[string]string
	me      string
	queries []url.Values
}

func newFake(t *testing.T, pages map[string]string) *fakeJira {
	f := &fakeJira{pages: pages, me: `{"accountId":"me-1","displayName":"Duke"}`}
	auth := "Basic " + base64.StdEncoding.EncodeToString([]byte("duke@example.com:secret"))
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != auth {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/rest/api/3/myself":
			_, _ = w.Write([]byte(f.me))
		case "/rest/api/3/search/jql":
			f.queries = append(f.queries, r.URL.Query())
			body, ok := f.pages[r.URL.Query().Get("nextPageToken")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(body))
		default:
			key, ok := strings.CutPrefix(r.URL.Path, "/rest/api/3/issue/")
			key, ok2 := strings.CutSuffix(key, "/comment")
			if !ok || !ok2 || r.URL.Query().Get("orderBy") != "-created" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(f.commentsOf(key))
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeJira) commentsOf(key string) []byte {
	for _, body := range f.pages {
		var p struct {
			Issues []struct {
				Key    string `json:"key"`
				Fields struct {
					Comment json.RawMessage `json:"comment"`
				} `json:"fields"`
			} `json:"issues"`
		}
		_ = json.Unmarshal([]byte(body), &p)
		for _, i := range p.Issues {
			if i.Key == key {
				return i.Fields.Comment
			}
		}
	}
	return []byte(`{"comments":[]}`)
}

func (f *fakeJira) watcher(t *testing.T) *Watcher {
	w := New(f.URL, "duke@example.com", "secret", filepath.Join(t.TempDir(), "jira.json"))
	w.Now = func() time.Time { return now }
	w.me = "me-1"
	w.state.Cursor = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	return w
}

func paragraph(nodes ...string) string {
	return `{"type":"doc","version":1,"content":[{"type":"paragraph","content":[` + strings.Join(nodes, ",") + `]}]}`
}

func text(s string) string { return `{"type":"text","text":"` + s + `"}` }

func mention(id, name string) string {
	return `{"type":"mention","attrs":{"id":"` + id + `","text":"@` + name + `"}}`
}

func commentJSON(id, author, name, created, body string) string {
	return `{"id":"` + id + `","author":{"accountId":"` + author + `","displayName":"` + name + `"},"body":` + body + `,"created":"` + created + `"}`
}

func historyJSON(id, author, name, created, items string) string {
	return `{"id":"` + id + `","author":{"accountId":"` + author + `","displayName":"` + name + `"},"created":"` + created + `","items":[` + items + `]}`
}

func issueJSON(key, summary, comments, histories string) string {
	return `{"id":"1","key":"` + key + `","fields":{"summary":"` + summary + `","comment":{"comments":[` + comments + `]}},
		"changelog":{"histories":[` + histories + `]}}`
}

func page(next string, issues ...string) string {
	tail := `"isLast":true`
	if next != "" {
		tail = `"nextPageToken":"` + next + `","isLast":false`
	}
	return `{"issues":[` + strings.Join(issues, ",") + `],` + tail + `}`
}

func assertLines(t *testing.T, events []hark.Event, want ...string) {
	t.Helper()
	var got []string
	for _, e := range events {
		got = append(got, e.String())
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestIssueActivity(t *testing.T) {
	f := newFake(t, map[string]string{
		"": page("p2", issueJSON("PROJ-1", "Login quebrado",
			commentJSON("10", "ana", "Ana Lima", "2026-11-23T17:10:00.000-0300", paragraph(text("pode olhar o log?")))+","+
				commentJSON("11", "bob", "Bob", "2026-11-23T17:20:00.000-0300", paragraph(text("cc "), mention("me-1", "Duke"), text(" sobre o deploy")))+","+
				commentJSON("12", "me-1", "Duke", "2026-11-23T17:25:00.000-0300", paragraph(text("eu mesmo"))),
			historyJSON("200", "ana", "Ana Lima", "2026-11-23T17:30:00.000-0300",
				`{"field":"status","fromString":"In Progress","toString":"Done"},{"field":"labels","fromString":"","toString":"x"}`)+","+
				historyJSON("199", "me-1", "Duke", "2026-11-23T17:15:00.000-0300", `{"field":"status","fromString":"To Do","toString":"In Progress"}`))),
		"p2": page("", issueJSON("PROJ-2", "Nova tela", "",
			historyJSON("300", "carla", "Carla", "2026-11-23T17:05:00.000-0300", `{"field":"assignee","to":"me-1","toString":"Duke"}`)+","+
				historyJSON("301", "carla", "Carla", "2026-11-23T17:06:00.000-0300", `{"field":"assignee","to":"other","toString":"Zé"}`))),
	})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z jira jira.assigned PROJ-2 "Nova tela (assigned by Carla)" `+f.URL+`/browse/PROJ-2`,
		`2026-11-23T20:10:00Z jira jira.commented PROJ-1 "Ana Lima: pode olhar o log?" `+f.URL+`/browse/PROJ-1?focusedCommentId=10`,
		`2026-11-23T20:20:00Z jira jira.mentioned PROJ-1 "Bob: cc @Duke sobre o deploy" `+f.URL+`/browse/PROJ-1?focusedCommentId=11`,
		`2026-11-23T20:30:00Z jira jira.transitioned PROJ-1 "Login quebrado: In Progress to Done (Ana Lima)" `+f.URL+`/browse/PROJ-1`)

	q := f.queries[0]
	if q.Get("jql") != "(assignee = currentUser() OR reporter = currentUser() OR watcher = currentUser()) AND updated >= -62m ORDER BY updated ASC" ||
		q.Get("fields") != "summary" || q.Get("expand") != "changelog" || f.queries[1].Get("nextPageToken") != "p2" {
		t.Errorf("queries %v", f.queries)
	}
}

func TestActivityIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", issueJSON("PROJ-1", "S",
		commentJSON("2", "ana", "Ana", "2026-11-23T20:10:00.000+0000", paragraph(text("b")))+","+
			commentJSON("1", "ana", "Ana", "2026-11-23T20:10:00.000+0000", paragraph(text("a")))+","+
			commentJSON("0", "ana", "Ana", "2026-11-23T19:00:00.000+0000", paragraph(text("antes"))), ""))})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:10:00Z jira jira.commented PROJ-1 "Ana: b" `+f.URL+`/browse/PROJ-1?focusedCommentId=2`,
		`2026-11-23T20:10:00Z jira jira.commented PROJ-1 "Ana: a" `+f.URL+`/browse/PROJ-1?focusedCommentId=1`)
	assertLines(t, w.poll(context.Background()))
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	for name, pages := range map[string]map[string]string{
		"search unavailable":   {},
		"second page missing":  {"": page("gone")},
		"bad json":             {"": "nope"},
		"date not a string":    {"": page("", issueJSON("PROJ-1", "S", `{"id":"1","author":{},"body":{},"created":123}`, ""))},
		"date in other format": {"": page("", issueJSON("PROJ-1", "S", `{"id":"1","author":{},"body":{},"created":"23/11/2026"}`, ""))},
	} {
		t.Run(name, func(t *testing.T) {
			events := newFake(t, pages).watcher(t).poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" {
				t.Fatalf("want one error event, got %v", events)
			}
		})
	}
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		if events := New(api, "e", "t", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesAndSavesState(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", issueJSON("PROJ-1", "S",
		commentJSON("1", "ana", "Ana", "2026-11-23T20:10:00.000+0000", paragraph(text("oi"))), ""))})
	state := filepath.Join(t.TempDir(), "jira.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL+"/", "duke@example.com", "secret", state)
	w.Now = func() time.Time { return now }
	w.Interval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	if err := w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() }); err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z jira jira.commented PROJ-1 "Ana: oi" `+f.URL+`/browse/PROJ-1?focusedCommentId=1`)
	var saved struct {
		Cursor time.Time `json:"cursor"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Cursor.Equal(time.Date(2026, 11, 23, 20, 10, 0, 0, time.UTC)) {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackAndPollsAgain(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	w := New(f.URL, "duke@example.com", "secret", filepath.Join(t.TempDir(), "jira.json"))
	w.Now = func() time.Time { return now }
	w.Interval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		for {
			f.mu.Lock()
			n := len(f.queries)
			f.mu.Unlock()
			if n >= 2 {
				cancel()
				return
			}
			time.Sleep(time.Millisecond)
		}
	}()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if got := f.queries[0].Get("jql"); !strings.Contains(got, "updated >= -62m") {
		t.Errorf("jql %q", got)
	}
}

func TestWatchFailsWithoutLoginOrState(t *testing.T) {
	f := newFake(t, map[string]string{})
	if err := New(f.URL, "duke@example.com", "wrong", "").Watch(context.Background(), func(hark.Event) {}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Errorf("login: %v", err)
	}
	if err := New(f.URL, "duke@example.com", "secret", t.TempDir()).Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "duke@example.com", "secret", filepath.Join(dir, "jira.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	_ = w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() })
	if len(got) != 1 || got[0].Kind != "error" {
		t.Fatalf("got %v", got)
	}
}

func TestDoctor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	state := filepath.Join(t.TempDir(), "jira.json")
	checks := New(f.URL, "duke@example.com", "secret", state).Doctor(context.Background())
	var got []string
	for _, c := range checks {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "|") != "login: logged in as Duke|search: issues you follow are searchable|state: "+state {
		t.Errorf("got %v", got)
	}
	if !strings.Contains(f.queries[0].Get("jql"), "currentUser()") || f.queries[0].Get("maxResults") != "1" {
		t.Errorf("search query %v", f.queries[0])
	}

	if checks := New(f.URL, "duke@example.com", "wrong", state).Doctor(context.Background()); len(checks) != 1 || checks[0].Err == nil {
		t.Errorf("bad login: %+v", checks)
	}
	noSearch := newFake(t, map[string]string{})
	if checks := New(noSearch.URL, "duke@example.com", "secret", state).Doctor(context.Background()); len(checks) != 3 || checks[1].Err == nil {
		t.Errorf("search failure not reported: %+v", checks)
	}
}

func TestSinceOverridesTheSavedCursor(t *testing.T) {
	f := newFake(t, map[string]string{})
	state := filepath.Join(t.TempDir(), "jira.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"PROJ-1:comment:1"}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "duke@example.com", "secret", state)
	w.Now = func() time.Time { return now }
	w.Since = now.Add(-3 * time.Hour)
	ctx, cancel := context.WithCancel(context.Background())
	if err := w.Watch(ctx, func(hark.Event) { cancel() }); err != nil {
		t.Fatal(err)
	}
	if got := f.queries[0].Get("jql"); !strings.Contains(got, "updated >= -182m") {
		t.Errorf("jql %q", got)
	}
	if !w.state.Cursor.Equal(w.Since) || w.state.Seen != nil {
		t.Errorf("state %+v", w.state)
	}
}
