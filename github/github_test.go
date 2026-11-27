package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dukex/hark"
)

var prExtras = regexp.MustCompile(`/pulls/\d+/(reviews|comments)$`)

var start = time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC)

type fakeGitHub struct {
	*httptest.Server
	mu         sync.Mutex
	routes     map[string]string
	hits       map[string]int
	query      string
	queries    map[string]string
	marked     []string
	markStatus int
}

func newFake(t *testing.T, routes map[string]string) *fakeGitHub {
	f := &fakeGitHub{routes: routes, hits: map[string]int{}, queries: map[string]string{}, markStatus: http.StatusResetContent}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		f.hits[r.URL.Path]++
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "bad credentials", http.StatusUnauthorized)
			return
		}
		if id, ok := strings.CutPrefix(r.URL.Path, "/notifications/threads/"); ok && r.Method == http.MethodPatch {
			f.marked = append(f.marked, id)
			w.WriteHeader(f.markStatus)
			return
		}
		if r.URL.Path == "/notifications" {
			f.query = r.URL.RawQuery
		}
		f.queries[r.URL.Path] = r.URL.RawQuery
		body, ok := f.routes[r.URL.Path]
		if !ok && prExtras.MatchString(r.URL.Path) {
			body, ok = "[]", true
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.ReplaceAll(body, "{{API}}", f.URL)))
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
	w.started = start
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
	repo := strings.Join(strings.Split(subjectPath, "/")[2:4], "/")
	return `{"id":"` + id + `","reason":"` + reason + `","updated_at":"` + updated + `","last_read_at":null,
		"repository":{"full_name":"` + repo + `"},
		"subject":{"type":"` + kind + `","url":"{{API}}` + subjectPath + `","latest_comment_url":` + latest + `}}`
}

func readAt(thread, at string) string {
	return strings.Replace(thread, `"last_read_at":null`, `"last_read_at":"`+at+`"`, 1)
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

func TestHandledThreadsAreMarkedRead(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		thread("2", "subscribed", "Release", "/repos/dukex/foo/releases/1", "", "2026-11-23T10:06:00Z") + `,` +
		thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	f := newFake(t, routes)

	f.watcher().poll(context.Background())
	if strings.Join(f.marked, ",") != "1" {
		t.Errorf("marked %v, want only the issue thread", f.marked)
	}
}

func TestMarkReadFailureBecomesErrorEvent(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	f := newFake(t, routes)
	f.markStatus = http.StatusForbidden

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:05:00Z github issue.updated dukex/foo#3 "author" u`,
		`2026-11-23T11:00:00Z github error "PATCH /notifications/threads/1: 403 Forbidden"`)
}

func TestOnlyUnreadFromTheLastHourAreRequested(t *testing.T) {
	f := newFake(t, baseRoutes())
	f.watcher().poll(context.Background())
	if f.query != "since=2026-11-23T09%3A00%3A00Z" {
		t.Errorf("query %q", f.query)
	}
}

func TestCommentAlreadyReadIsIgnored(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + readAt(thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T10:05:00Z"), "2026-11-23T09:30:00Z") + `]`
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
			routes["/users/dukex/events"] = `[]`
			f := newFake(t, routes)
			events := f.watcher().poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" {
				t.Fatalf("want one error event, got %v", lines(events))
			}
			if len(f.marked) != 0 {
				t.Errorf("failed thread marked read: %v", f.marked)
			}
		})
	}
}

func TestUnreachableAPIBecomesErrorEvent(t *testing.T) {
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		w := New(api, "secret")
		events := w.poll(context.Background())
		if len(events) != 2 || events[0].Kind != "error" || events[1].Kind != "error" {
			t.Fatalf("%s: want an error event per source, got %v", api, lines(events))
		}
	}
}

func TestUnsupportedSubjectsAreSkipped(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "subscribed", "Release", "/repos/dukex/foo/releases/1", "", "2026-11-23T10:05:00Z") + `]`
	f := newFake(t, routes)
	assertLines(t, f.watcher().poll(context.Background()))
}

func TestMyIssueClosed(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:07:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"closed","user":{"login":"dukex"},"closed_at":"2026-11-23T10:07:00Z","closed_by":{"login":"carol"},"html_url":"https://github.com/dukex/foo/issues/3"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:07:00Z github issue.closed dukex/foo#3 @carol https://github.com/dukex/foo/issues/3`)
}

func TestMyPRMergedAndClosed(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		thread("2", "author", "PullRequest", "/repos/dukex/foo/pulls/13", "", "2026-11-23T10:09:00Z") + `,` +
		thread("1", "author", "PullRequest", "/repos/dukex/foo/pulls/12", "", "2026-11-23T10:08:00Z") + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"state":"closed","merged":true,"merged_by":{"login":"bob"},"user":{"login":"dukex"},"closed_at":"2026-11-23T10:08:00Z","html_url":"https://github.com/dukex/foo/pull/12"}`
	routes["/repos/dukex/foo/pulls/13"] = `{"number":13,"state":"closed","merged":false,"user":{"login":"dukex"},"closed_at":"2026-11-23T10:09:00Z","html_url":"https://github.com/dukex/foo/pull/13"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:08:00Z github pr.merged dukex/foo#12 @bob https://github.com/dukex/foo/pull/12`,
		`2026-11-23T10:09:00Z github pr.closed dukex/foo#13 https://github.com/dukex/foo/pull/13`)
}

func TestClosedWithCommentReportsBoth(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T10:07:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"closed","user":{"login":"dukex"},"closed_at":"2026-11-23T10:07:00Z","closed_by":{"login":"carol"},"html_url":"u"}`
	routes["/repos/dukex/foo/issues/comments/9"] = `{"user":{"login":"carol"},"body":"resolvido","created_at":"2026-11-23T10:06:59Z","html_url":"c9"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:06:59Z github issue.commented dukex/foo#3 @carol "resolvido" c9`,
		`2026-11-23T10:07:00Z github issue.closed dukex/foo#3 @carol u`)
}

func TestCloseAlreadyReadIsIgnored(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + readAt(thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:07:00Z"), "2026-11-21T00:00:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"state":"closed","user":{"login":"dukex"},"closed_at":"2026-11-20T00:00:00Z","html_url":"u"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:07:00Z github issue.updated dukex/foo#3 "author" u`)
}

func activity(id, typ, action, created, payload string) string {
	return `{"id":"` + id + `","type":"` + typ + `","created_at":"` + created + `","actor":{"login":"dukex"},
		"repo":{"name":"dukex/foo"},"payload":{"action":"` + action + `",` + payload + `}}`
}

func TestMyOwnActions(t *testing.T) {
	routes := baseRoutes()
	routes["/users/dukex/events"] = `[` +
		activity("105", "IssuesEvent", "closed", "2026-11-23T10:05:00Z", `"issue":{"number":3,"url":"{{API}}/repos/dukex/foo/issues/3"}`) + `,` +
		activity("104", "PullRequestEvent", "closed", "2026-11-23T10:04:00Z", `"number":11,"pull_request":{"number":11,"url":"{{API}}/repos/dukex/foo/pulls/11"}`) + `,` +
		activity("103", "PullRequestEvent", "closed", "2026-11-23T10:03:00Z", `"number":12,"pull_request":{"number":12,"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `,` +
		activity("102", "PullRequestEvent", "opened", "2026-11-23T10:02:00Z", `"number":12,"pull_request":{"number":12,"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `,` +
		activity("101", "PullRequestEvent", "labeled", "2026-11-23T10:01:30Z", `"number":12,"pull_request":{"number":12,"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `,` +
		activity("100", "PushEvent", "", "2026-11-23T10:01:00Z", `"ref":"main"`) + `,` +
		activity("99", "PullRequestEvent", "opened", "2026-11-23T09:00:00Z", `"number":10,"pull_request":{"number":10,"url":"{{API}}/repos/dukex/foo/pulls/10"}`) + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"title":"bug","html_url":"https://github.com/dukex/foo/issues/3"}`
	routes["/repos/dukex/foo/pulls/11"] = `{"number":11,"title":"wip","merged":false,"html_url":"https://github.com/dukex/foo/pull/11"}`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"title":"feat","merged":true,"html_url":"https://github.com/dukex/foo/pull/12"}`
	f := newFake(t, routes)
	w := f.watcher()

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T10:02:00Z github pr.opened dukex/foo#12 @dukex "feat" https://github.com/dukex/foo/pull/12`,
		`2026-11-23T10:03:00Z github pr.merged dukex/foo#12 @dukex "feat" https://github.com/dukex/foo/pull/12`,
		`2026-11-23T10:04:00Z github pr.closed dukex/foo#11 @dukex "wip" https://github.com/dukex/foo/pull/11`,
		`2026-11-23T10:05:00Z github issue.closed dukex/foo#3 @dukex "bug" https://github.com/dukex/foo/issues/3`)
	assertLines(t, w.poll(context.Background()))
}

func TestMyMergesAndReviewsAsGitHubSendsThemNow(t *testing.T) {
	pr := func(n string) string {
		return `"number":` + n + `,"pull_request":{"number":` + n + `,"url":"{{API}}/repos/dukex/foo/pulls/` + n + `"}`
	}
	review := func(state, n string) string {
		return `"review":{"id":1,"state":"` + state + `","body":null,"html_url":"https://github.com/dukex/foo/pull/` + n + `#pullrequestreview-1"},` + pr(n)
	}
	routes := baseRoutes()
	routes["/users/dukex/events"] = `[` +
		activity("28000000002", "PushEvent", "", "2026-11-23T09:30:00Z", `"ref":"main"`) + `,` +
		activity("21000000006", "PullRequestReviewEvent", "dismissed", "2026-11-23T10:06:00Z", review("dismissed", "14")) + `,` +
		activity("21000000005", "PullRequestReviewEvent", "created", "2026-11-23T10:05:00Z", review("commented", "14")) + `,` +
		activity("21000000004", "PullRequestReviewEvent", "created", "2026-11-23T10:04:00Z", review("changes_requested", "14")) + `,` +
		activity("28000000001", "PushEvent", "", "2026-11-23T10:03:30Z", `"ref":"main"`) + `,` +
		activity("21000000003", "PullRequestReviewEvent", "created", "2026-11-23T10:03:00Z", review("approved", "13")) + `,` +
		activity("21000000002", "PullRequestEvent", "merged", "2026-11-23T10:02:00Z", pr("12")) + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"title":"feat","merged":true,"html_url":"https://github.com/dukex/foo/pull/12"}`
	routes["/repos/dukex/foo/pulls/13"] = `{"number":13,"title":"fix","html_url":"https://github.com/dukex/foo/pull/13"}`
	routes["/repos/dukex/foo/pulls/14"] = `{"number":14,"title":"wip","html_url":"https://github.com/dukex/foo/pull/14"}`
	f := newFake(t, routes)
	w := f.watcher()

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T10:02:00Z github pr.merged dukex/foo#12 @dukex "feat" https://github.com/dukex/foo/pull/12`,
		`2026-11-23T10:03:00Z github pr.approved dukex/foo#13 @dukex "fix" https://github.com/dukex/foo/pull/13#pullrequestreview-1`,
		`2026-11-23T10:04:00Z github pr.changes_requested dukex/foo#14 @dukex "wip" https://github.com/dukex/foo/pull/14#pullrequestreview-1`)

	f.set("/users/dukex/events", `[`+
		activity("21000000007", "PullRequestEvent", "merged", "2026-11-23T10:07:00Z", pr("13"))+`,`+
		strings.TrimPrefix(routes["/users/dukex/events"], `[`))
	assertLines(t, w.poll(context.Background()),
		`2026-11-23T10:07:00Z github pr.merged dukex/foo#13 @dukex "fix" https://github.com/dukex/foo/pull/13`)
}

func TestOwnEventItemUnavailableBecomesErrorEvent(t *testing.T) {
	routes := baseRoutes()
	routes["/users/dukex/events"] = `[` + activity("1", "PullRequestEvent", "opened", "2026-11-23T10:02:00Z", `"pull_request":{"url":"{{API}}/repos/dukex/foo/pulls/404"}`) + `]`
	f := newFake(t, routes)
	events := f.watcher().poll(context.Background())
	if len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("want one error event, got %v", lines(events))
	}
}

func TestOwnEventsUnavailableBecomesErrorEvent(t *testing.T) {
	routes := baseRoutes()
	delete(routes, "/users/dukex/events")
	f := newFake(t, routes)
	events := f.watcher().poll(context.Background())
	if len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("want one error event, got %v", lines(events))
	}
}

func TestWatchedThreadsAreMarked(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		thread("2", "subscribed", "PullRequest", "/repos/acme/lib/pulls/40", "", "2026-11-23T10:11:00Z") + `,` +
		thread("1", "mention", "Issue", "/repos/acme/lib/issues/7", "/repos/acme/lib/issues/comments/70", "2026-11-23T10:10:00Z") + `]`
	routes["/repos/acme/lib/issues/7"] = `{"number":7,"state":"open","user":{"login":"erin"},"html_url":"https://github.com/acme/lib/issues/7"}`
	routes["/repos/acme/lib/issues/comments/70"] = `{"user":{"login":"frank"},"body":"@dukex olha isso","created_at":"2026-11-23T10:10:00Z","html_url":"c70"}`
	routes["/repos/acme/lib/pulls/40"] = `{"number":40,"state":"closed","merged":true,"merged_by":{"login":"erin"},"user":{"login":"erin"},"closed_at":"2026-11-23T10:11:00Z","html_url":"https://github.com/acme/lib/pull/40"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:10:00Z github issue.commented acme/lib#7 @frank watching "@dukex olha isso" c70`,
		`2026-11-23T10:11:00Z github pr.merged acme/lib#40 @erin watching https://github.com/acme/lib/pull/40`)
}

func ciThread(id, reason, kind, title, updated string) string {
	return `{"id":"` + id + `","reason":"` + reason + `","updated_at":"` + updated + `","last_read_at":null,
		"repository":{"full_name":"acme/api"},
		"subject":{"type":"` + kind + `","title":"` + title + `","url":null,"latest_comment_url":null}}`
}

func TestCINotifications(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		ciThread("6", "ci_activity", "CheckSuite", "Terragrunt workflow run, Attempt #5 failed for main branch", "2026-11-23T10:06:00Z") + `,` +
		ciThread("5", "ci_activity", "CheckSuite", "Lint run for main", "2026-11-23T10:05:00Z") + `,` +
		ciThread("4", "approval_requested", "WorkflowRun", "erin requested your review to deploy to an environment", "2026-11-23T10:04:00Z") + `,` +
		ciThread("3", "ci_activity", "CheckSuite", "AI Factory workflow run skipped for feat/x branch", "2026-11-23T10:03:00Z") + `,` +
		ciThread("2", "ci_activity", "CheckSuite", "CI/CD workflow run succeeded for main branch", "2026-11-23T10:02:00Z") + `,` +
		ciThread("1", "ci_activity", "CheckSuite", "CI/CD workflow run failed for fix/y branch", "2026-11-23T10:01:00Z") + `]`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:01:00Z github ci.failed acme/api "CI/CD workflow run failed for fix/y branch"`,
		`2026-11-23T10:02:00Z github ci.succeeded acme/api "CI/CD workflow run succeeded for main branch"`,
		`2026-11-23T10:03:00Z github ci.skipped acme/api "AI Factory workflow run skipped for feat/x branch"`,
		`2026-11-23T10:04:00Z github ci.approval_requested acme/api "erin requested your review to deploy to an environment"`,
		`2026-11-23T10:05:00Z github ci.ci_activity acme/api "Lint run for main"`,
		`2026-11-23T10:06:00Z github ci.failed acme/api "Terragrunt workflow run, Attempt #5 failed for main branch"`)
	if strings.Join(f.marked, ",") != "1,2,3,4,5,6" {
		t.Errorf("marked %v", f.marked)
	}
}

func TestNeverReadThreadIgnoresActivityBeforeTheWindow(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "review_requested", "PullRequest", "/repos/acme/lib/pulls/126", "/repos/acme/lib/issues/comments/5", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/acme/lib/pulls/126"] = `{"number":126,"state":"closed","merged":true,"merged_by":{"login":"erin"},"user":{"login":"erin"},"closed_at":"2026-11-04T19:51:38Z","html_url":"https://github.com/acme/lib/pull/126"}`
	routes["/repos/acme/lib/issues/comments/5"] = `{"user":{"login":"frank"},"body":"velho","created_at":"2026-11-04T19:00:00Z","html_url":"c5"}`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:05:00Z github pr.updated acme/lib#126 watching "review_requested" https://github.com/acme/lib/pull/126`)
}

func TestDoctor(t *testing.T) {
	f := newFake(t, baseRoutes())
	assertChecks(t, New(f.URL, "secret").Doctor(context.Background()),
		"token: logged in as dukex", "notifications: readable")

	delete(f.routes, "/notifications")
	checks := New(f.URL, "secret").Doctor(context.Background())
	if len(checks) != 2 || checks[1].Err == nil {
		t.Errorf("notifications failure not reported: %+v", checks)
	}
	checks = New(f.URL, "wrong").Doctor(context.Background())
	if len(checks) != 1 || checks[0].Name != "token" || checks[0].Err == nil {
		t.Errorf("bad token: %+v", checks)
	}
}

func assertChecks(t *testing.T, checks []hark.Check, want ...string) {
	t.Helper()
	var got []string
	for _, c := range checks {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("\ngot:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

func TestReviewsOnMyPR(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "PullRequest", "/repos/acme/app/pulls/7", "", "2026-11-23T10:30:00Z") + `]`
	routes["/repos/acme/app/pulls/7"] = `{"number":7,"state":"open","user":{"login":"dukex"},"html_url":"https://github.com/acme/app/pull/7"}`
	routes["/repos/acme/app/pulls/7/reviews"] = `[
		{"id":1,"user":{"login":"rita"},"state":"COMMENTED","body":"antiga","submitted_at":"2026-11-23T08:00:00Z","html_url":"r1"},
		{"id":2,"user":{"login":"review-bot[bot]"},"state":"COMMENTED","body":"<!-- review-bot -->\n<!-- pass=review -->\n**Review** found 2 issues","submitted_at":"2026-11-23T10:05:00Z","html_url":"r2"},
		{"id":3,"user":{"login":"dukex"},"state":"COMMENTED","body":"","submitted_at":"2026-11-23T10:10:00Z","html_url":"r3"},
		{"id":4,"user":{"login":"rita"},"state":"COMMENTED","body":"","submitted_at":"2026-11-23T10:15:00Z","html_url":"r4"},
		{"id":5,"user":{"login":"sam"},"state":"CHANGES_REQUESTED","body":"falta teste","submitted_at":"2026-11-23T10:20:00Z","html_url":"r5"},
		{"id":6,"user":{"login":"sam"},"state":"APPROVED","body":"","submitted_at":"2026-11-23T10:25:00Z","html_url":"r6"}]`
	routes["/repos/acme/app/pulls/7/comments"] = `[
		{"pull_request_review_id":2,"body":"[1] token","created_at":"2026-11-23T10:05:00Z"},
		{"pull_request_review_id":2,"body":"[2] botões","created_at":"2026-11-23T10:05:00Z"},
		{"pull_request_review_id":4,"body":"Se o token demora, a tela fica vazia","created_at":"2026-11-23T10:15:00Z"},
		{"pull_request_review_id":4,"body":"Detalhe, não bloqueia","created_at":"2026-11-23T10:15:00Z"}]`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:05:00Z github pr.reviewed acme/app#7 @review-bot[bot] "**Review** found 2 issues (2 inline comments)" r2`,
		`2026-11-23T10:15:00Z github pr.reviewed acme/app#7 @rita "Se o token demora, a tela fica vazia (2 inline comments)" r4`,
		`2026-11-23T10:20:00Z github pr.changes_requested acme/app#7 @sam "falta teste" r5`,
		`2026-11-23T10:25:00Z github pr.approved acme/app#7 @sam r6`)
	if q := f.queries["/repos/acme/app/pulls/7/comments"]; q != "per_page=100&since=2026-11-23T09%3A00%3A00Z" {
		t.Errorf("review comments query %q", q)
	}
}

func TestInlineLatestCommentIsCoveredByItsReview(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "PullRequest", "/repos/acme/app/pulls/7", "/repos/acme/app/pulls/comments/40", "2026-11-23T10:15:00Z") + `]`
	routes["/repos/acme/app/pulls/7"] = `{"number":7,"user":{"login":"dukex"},"html_url":"u"}`
	routes["/repos/acme/app/pulls/7/reviews"] = `[{"id":4,"user":{"login":"rita"},"state":"COMMENTED","body":"ok","submitted_at":"2026-11-23T10:15:00Z","html_url":"r4"}]`
	f := newFake(t, routes)

	assertLines(t, f.watcher().poll(context.Background()),
		`2026-11-23T10:15:00Z github pr.reviewed acme/app#7 @rita "ok" r4`)
}

func TestReviewFailuresBecomeErrorEvents(t *testing.T) {
	for _, broken := range []string{"/repos/acme/app/pulls/7/reviews", "/repos/acme/app/pulls/7/comments"} {
		routes := baseRoutes()
		routes["/notifications"] = `[` + thread("1", "author", "PullRequest", "/repos/acme/app/pulls/7", "", "2026-11-23T10:15:00Z") + `]`
		routes["/repos/acme/app/pulls/7"] = `{"number":7,"user":{"login":"dukex"},"html_url":"u"}`
		routes["/repos/acme/app/pulls/7/reviews"] = `[{"id":4,"user":{"login":"rita"},"state":"COMMENTED","body":"ok","submitted_at":"2026-11-23T10:15:00Z","html_url":"r4"}]`
		routes[broken] = "nope"
		f := newFake(t, routes)
		events := f.watcher().poll(context.Background())
		if len(events) != 1 || events[0].Kind != "error" || len(f.marked) != 0 {
			t.Errorf("%s: events %v, marked %v", broken, lines(events), f.marked)
		}
	}
}

func TestSinceReplacesTheStartupWindow(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` + thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "/repos/dukex/foo/issues/comments/9", "2026-11-23T07:30:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	routes["/repos/dukex/foo/issues/comments/9"] = `{"user":{"login":"alice"},"body":"still valid?","created_at":"2026-11-23T07:30:00Z","html_url":"c9"}`
	routes["/users/dukex/events"] = `[` +
		activity("2", "PullRequestEvent", "opened", "2026-11-23T07:45:00Z", `"pull_request":{"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `,` +
		activity("1", "PullRequestEvent", "opened", "2026-11-23T06:59:00Z", `"pull_request":{"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"title":"feat","html_url":"p12"}`
	f := newFake(t, routes)
	w := f.watcher()
	w.Since = time.Date(2026, 11, 23, 7, 0, 0, 0, time.UTC)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T07:30:00Z github issue.commented dukex/foo#3 @alice "still valid?" c9`,
		`2026-11-23T07:45:00Z github pr.opened dukex/foo#12 @dukex "feat" p12`)
	if f.query != "since=2026-11-23T07%3A00%3A00Z" {
		t.Errorf("query %q", f.query)
	}
}
