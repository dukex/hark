package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dukex/hark"
)

var start = time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)

type fakeGitHub struct {
	*httptest.Server
	mu     sync.Mutex
	routes map[string]string
	hits   map[string]int
}

func newFake(t *testing.T, routes map[string]string) *fakeGitHub {
	f := &fakeGitHub{routes: routes, hits: map[string]int{}}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.hits[r.URL.Path]++
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad credentials", http.StatusUnauthorized)
			return
		}
		body, ok := f.routes[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(strings.ReplaceAll(body, "{{API}}", f.URL)))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeGitHub) set(path, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.routes[path] = body
}

func (f *fakeGitHub) watcher() *Watcher {
	w := New(f.URL, "secret")
	w.me = "dukex"
	w.since = start
	w.Now = func() time.Time { return start.Add(time.Hour) }
	return w
}

func baseRoutes() map[string]string {
	return map[string]string{
		"/user":               `{"login":"dukex"}`,
		"/notifications":      `[]`,
		"/users/dukex/events": `[]`,
	}
}

func thread(id, reason, kind, subjectPath, commentPath, updated string) string {
	latest := "null"
	if commentPath != "" {
		latest = `"{{API}}` + commentPath + `"`
	}
	return `{"id":"` + id + `","reason":"` + reason + `","updated_at":"` + updated + `",
		"repository":{"full_name":"dukex/foo"},
		"subject":{"type":"` + kind + `","url":"{{API}}` + subjectPath + `","latest_comment_url":` + latest + `}}`
}

func lines(events []hark.Event) []string {
	var out []string
	for _, e := range events {
		out = append(out, e.String())
	}
	return out
}

func assertLines(t *testing.T, events []hark.Event, want ...string) {
	t.Helper()
	got := lines(events)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestCommentOnMyIssue(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"open","user":{"login":"dukex"},"created_at":"2026-11-01T00:00:00Z","html_url":"https://github.com/dukex/foo/issues/3"}`
	routes["/repos/dukex/foo/issues/comments/9"] = `{"user":{"login":"alice"},"body":"funciona aqui","created_at":"2026-11-23T10:05:00Z","html_url":"https://github.com/dukex/foo/issues/3#issuecomment-9"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:05:00Z github issue.commented dukex/foo#3 @alice "funciona aqui" https://github.com/dukex/foo/issues/3#issuecomment-9`)
}

func TestCommentOnMyPR(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("2", "author", "PullRequest", "/repos/dukex/foo/pulls/12", "/repos/dukex/foo/issues/comments/7", "2026-11-23T10:06:00Z") + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"state":"open","user":{"login":"dukex"},"created_at":"2026-11-01T00:00:00Z","html_url":"https://github.com/dukex/foo/pull/12"}`
	routes["/repos/dukex/foo/issues/comments/7"] = `{"user":{"login":"bob"},"body":"lgtm","created_at":"2026-11-23T10:06:00Z","html_url":"https://github.com/dukex/foo/pull/12#issuecomment-7"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:06:00Z github pr.commented dukex/foo#12 @bob "lgtm" https://github.com/dukex/foo/pull/12#issuecomment-7`)
}

func TestCommentIsReportedOnce(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"open","user":{"login":"dukex"},"created_at":"2026-11-01T00:00:00Z","html_url":"u"}`
	routes["/repos/dukex/foo/issues/comments/9"] = `{"user":{"login":"alice"},"body":"oi","created_at":"2026-11-23T10:05:00Z","html_url":"c9"}`
	f := newFake(t, routes)
	w := f.watcher()

	w.poll(context.Background())
	assertLines(t, w.poll(context.Background()))
}

func TestCommentFromBeforeStartIsIgnored(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"open","user":{"login":"dukex"},"created_at":"2026-11-01T00:00:00Z","html_url":"u"}`
	routes["/repos/dukex/foo/issues/comments/9"] = `{"user":{"login":"alice"},"body":"velho","created_at":"2026-11-23T09:00:00Z","html_url":"c9"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:05:00Z github issue.updated dukex/foo#3 "author" u`)
}

func TestWatchFailsWhenLoginIsRejected(t *testing.T) {
	f := newFake(t, baseRoutes())
	w := New(f.URL, "wrong")
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("want 401 error, got %v", err)
	}
}

func TestWatchPollsUntilCanceled(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	f := newFake(t, routes)
	w := New(f.URL+"/", "secret")
	w.Interval = time.Millisecond
	w.Now = func() time.Time { return start }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	err := w.Watch(ctx, func(e hark.Event) {
		got = append(got, e)
		f.set("/notifications", `[`+thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:09:00Z")+`]`)
		if len(got) == 2 {
			cancel()
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, got,
		`2026-11-23T10:05:00Z github issue.updated dukex/foo#3 "author" u`,
		`2026-11-23T10:09:00Z github issue.updated dukex/foo#3 "author" u`)
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	cases := map[string]map[string]string{
		"notifications unavailable": {"/notifications": `not json`},
		"subject unavailable":       {"/notifications": `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/404", "", "2026-11-23T10:05:00Z") + `]`},
		"comment unavailable": {
			"/notifications":            `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/404", "2026-11-23T10:05:00Z") + `]`,
			"/repos/dukex/foo/issues/3": `{"number":3}`,
		},
	}
	for name, routes := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFake(t, routes)
			events := f.watcher().poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" {
				t.Fatalf("want one error event, got %v", lines(events))
			}
		})
	}
}

func TestUnreachableAPIBecomesErrorEvent(t *testing.T) {
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		w := New(api, "secret")
		if events := w.poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: want one error event, got %v", api, lines(events))
		}
	}
}

func TestUnsupportedSubjectsAreSkipped(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "subscribed", "Release", "/repos/dukex/foo/releases/1", "", "2026-11-23T10:05:00Z") + `]`
	f := newFake(t, routes)
	assertLines(t, f.watcher().poll(context.Background()))
}
