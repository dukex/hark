package linear

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dukex/hark"
)

var now = time.Date(2026, 11, 23, 21, 0, 0, 0, time.UTC)

type request struct {
	Query     string         `json:"query"`
	Variables map[string]any `json:"variables"`
}

type fakeLinear struct {
	*httptest.Server
	mu       sync.Mutex
	pages    map[string]string
	requests []request
}

func newFake(t *testing.T, pages map[string]string) *fakeLinear {
	f := &fakeLinear{pages: pages}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Method != http.MethodPost || r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if r.Header.Get("Authorization") != "lin_api_1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Authentication required, not authenticated"}]}`))
			return
		}
		var req request
		_ = json.NewDecoder(r.Body).Decode(&req)
		if strings.Contains(req.Query, "viewer") {
			_, _ = w.Write([]byte(`{"data":{"viewer":{"name":"Duke"}}}`))
			return
		}
		f.requests = append(f.requests, req)
		if filter, _ := json.Marshal(req.Variables["filter"]); strings.Contains(string(filter), "bad") {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Argument Validation Error"}]}`))
			return
		}
		after, _ := req.Variables["after"].(string)
		body, ok := f.pages[after]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeLinear) watcher(t *testing.T) *Watcher {
	w := New(f.URL, "lin_api_1", filepath.Join(t.TempDir(), "linear.json"))
	w.Now = func() time.Time { return now }
	w.state.Cursor = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	return w
}

func (f *fakeLinear) asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.requests)
}

func (f *fakeLinear) filter(i int) string {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, _ := json.Marshal(f.requests[i].Variables["filter"])
	return string(b)
}

func at(clock string) string { return "2026-11-23T" + clock + ".000Z" }

func issueURL(key string) string { return "https://linear.app/acme/issue/" + key + "/title" }

func issue(key, title, status string) string {
	return fmt.Sprintf(`{"identifier":%q,"title":%q,"url":%q,"state":{"name":%q}}`, key, title, issueURL(key), status)
}

func comment(key, id, body string) string {
	return fmt.Sprintf(`{"body":%q,"url":%q}`, body, issueURL(key)+"#comment-"+id)
}

func node(id, kind, clock, actor, issue, comment string) string {
	fields := fmt.Sprintf(`"id":%q,"type":%q,"createdAt":%q`, id, kind, at(clock))
	if actor != "" {
		fields += fmt.Sprintf(`,"actor":{"displayName":%q}`, actor)
	}
	if issue != "" {
		fields += `,"issue":` + issue
	}
	if comment != "" {
		fields += `,"comment":` + comment
	}
	return "{" + fields + "}"
}

func page(next string, nodes ...string) string {
	return fmt.Sprintf(`{"data":{"notifications":{"nodes":[%s],"pageInfo":{"hasNextPage":%t,"endCursor":%q}}}}`, strings.Join(nodes, ","), next != "", next)
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

func TestNotificationsBecomeEvents(t *testing.T) {
	f := newFake(t, map[string]string{
		"": page("c1",
			node("n7", "issueNewComment", "20:50:00", "ana", issue("ENG-1", "Fix login", "In Progress"), comment("ENG-1", "c7", "pushed a fix")),
			node("n6", "issueStatusChanged", "20:40:00", "bob", issue("ENG-2", "Ship v2", "Done"), ""),
			node("n5", "issueAssignedToYou", "20:30:00", "carol", issue("ENG-3", "Refund copy", "Todo"), "")),
		"c1": page("",
			node("n4", "issueCommentMention", "20:20:00", "Ana Silva", issue("ENG-1", "Fix login", "In Progress"), comment("ENG-1", "c4", "@duke can you check?")),
			node("n3", "issueMention", "20:15:00", "dan", issue("ENG-4", "Onboarding", "Backlog"), ""),
			node("n2", "issueEmojiReaction", "20:10:00", "erin", issue("ENG-4", "Onboarding", "Backlog"), ""),
			node("n1", "projectUpdateCreated", "20:05:00", "", "", "")),
	})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z linear linear.project_update_created`,
		`2026-11-23T20:10:00Z linear linear.emoji_reaction ENG-4 @erin "Onboarding" `+issueURL("ENG-4"),
		`2026-11-23T20:15:00Z linear linear.mentioned ENG-4 @dan "Onboarding" `+issueURL("ENG-4"),
		`2026-11-23T20:20:00Z linear linear.mentioned ENG-1 @AnaSilva "@duke can you check?" `+issueURL("ENG-1")+"#comment-c4",
		`2026-11-23T20:30:00Z linear linear.assigned ENG-3 @carol "Refund copy" `+issueURL("ENG-3"),
		`2026-11-23T20:40:00Z linear linear.transitioned ENG-2 @bob "Ship v2: Done" `+issueURL("ENG-2"),
		`2026-11-23T20:50:00Z linear linear.commented ENG-1 @ana "pushed a fix" `+issueURL("ENG-1")+"#comment-c7")

	if got := f.filter(0); got != `{"createdAt":{"gte":"2026-11-23T20:00:00Z"}}` {
		t.Errorf("filter %s", got)
	}
	if f.requests[1].Variables["after"] != "c1" || f.requests[0].Variables["first"] != float64(100) {
		t.Errorf("paging %v", f.requests[1].Variables)
	}
	if !strings.Contains(f.requests[0].Query, "includeArchived: true") {
		t.Error("archived notifications are skipped")
	}
}

func TestOverlapIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		node("n2", "issueMention", "20:58:00", "ana", issue("ENG-1", "Late", "Todo"), ""),
		node("n1", "issueMention", "20:10:00", "ana", issue("ENG-1", "Old", "Todo"), ""))})
	w := f.watcher(t)

	if events := w.poll(context.Background()); len(events) != 2 {
		t.Fatalf("got %v", events)
	}
	if !w.state.Cursor.Equal(now.Add(-lag)) || strings.Join(w.state.Seen, ",") != "n2" {
		t.Errorf("state %+v", w.state)
	}
	assertLines(t, w.poll(context.Background()))

	w.state.Cursor = now.Add(-time.Minute)
	w.poll(context.Background())
	if !w.state.Cursor.Equal(now.Add(-time.Minute)) {
		t.Errorf("cursor moved back to %v", w.state.Cursor)
	}
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	for name, pages := range map[string]map[string]string{
		"not found":           {},
		"second page missing": {"": page("c1")},
		"graphql error":       {"": `{"errors":[{"message":"Rate limit exceeded"}]}`},
		"bad json":            {"": "nope"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newFake(t, pages).watcher(t)
			before := w.state.Cursor
			events := w.poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" || events[0].Provider != "linear" {
				t.Fatalf("want one error event, got %v", events)
			}
			if !w.state.Cursor.Equal(before) {
				t.Error("cursor moved after a failure")
			}
		})
	}
	if events := newFake(t, map[string]string{"": `{"errors":[{"message":"Rate limit exceeded"}]}`}).watcher(t).poll(context.Background()); events[0].Text != "Rate limit exceeded" {
		t.Errorf("text %q", events[0].Text)
	}
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		if events := New(api, "lin_api_1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesAndSavesState(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", node("n1", "issueAssignedToYou", "20:10:00", "ana", issue("ENG-1", "Fix login", "Todo"), ""))})
	state := filepath.Join(t.TempDir(), "linear.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "lin_api_1", state)
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	if err := w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() }); err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z linear linear.assigned ENG-1 @ana "Fix login" `+issueURL("ENG-1"))
	var saved struct {
		Cursor time.Time `json:"cursor"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Cursor.Equal(now.Add(-lag)) {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackAndPollsAgain(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	w := New(f.URL, "lin_api_1", filepath.Join(t.TempDir(), "linear.json"))
	w.Now = func() time.Time { return now }
	w.Interval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		for f.asked() < 2 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if got := f.filter(0); got != `{"createdAt":{"gte":"2026-11-23T20:00:00Z"}}` {
		t.Errorf("filter %s", got)
	}
}

func TestSinceOverridesTheSavedCursor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	state := filepath.Join(t.TempDir(), "linear.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"n1"}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "lin_api_1", state)
	w.Now = func() time.Time { return now }
	w.Since = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for f.asked() < 1 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if got := f.filter(0); got != `{"createdAt":{"gte":"2026-11-10T09:00:00Z"}}` {
		t.Errorf("filter %s", got)
	}
}

func TestFilterNarrowsTheQueryAndIsCheckedOnStart(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	w := f.watcher(t)
	w.Filter = `{"type":{"in":["issueMention"]}}`
	w.poll(context.Background())
	if got := f.filter(0); got != `{"and":[{"createdAt":{"gte":"2026-11-23T20:00:00Z"}},{"type":{"in":["issueMention"]}}]}` {
		t.Errorf("filter %s", got)
	}

	for filter, want := range map[string]string{
		`{"type":"bad"}`: `--filter "{\"type\":\"bad\"}": Argument Validation Error`,
		`type = issue`:   `--filter "type = issue": not a JSON object`,
		`[1]`:            `--filter "[1]": not a JSON object`,
	} {
		w = New(newFake(t, map[string]string{"": page("")}).URL, "lin_api_1", filepath.Join(t.TempDir(), "linear.json"))
		w.Filter = filter
		if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != want {
			t.Errorf("%s: err %v", filter, err)
		}
	}

	f = newFake(t, map[string]string{"": page("")})
	w = New(f.URL, "lin_api_1", filepath.Join(t.TempDir(), "linear.json"))
	w.Filter = `{"type":{"eq":"issueMention"}}`
	w.Interval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for f.asked() < 2 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	f.mu.Lock()
	probe := f.requests[0].Variables
	f.mu.Unlock()
	if probe["first"] != float64(1) || fmt.Sprint(probe["filter"]) != "map[type:map[eq:issueMention]]" {
		t.Errorf("probe %v", probe)
	}
}

func TestWatchFailsWithoutLoginOrState(t *testing.T) {
	f := newFake(t, nil)
	if err := New(f.URL, "wrong", "").Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != "Authentication required, not authenticated" {
		t.Errorf("login: %v", err)
	}
	if err := New(f.URL, "lin_api_1", t.TempDir()).Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "lin_api_1", filepath.Join(dir, "linear.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	_ = w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() })
	if len(got) != 1 || got[0].Kind != "error" {
		t.Fatalf("got %v", got)
	}
}

func TestDoctor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", node("n1", "issueMention", "20:10:00", "ana", issue("ENG-1", "Fix login", "Todo"), ""))})
	state := filepath.Join(t.TempDir(), "linear.json")
	w := New(f.URL, "lin_api_1", state)
	w.Now = func() time.Time { return now }
	var got []string
	for _, c := range w.Doctor(context.Background()) {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "|") != "api key: logged in as Duke|notifications: readable|state: "+state {
		t.Errorf("got %v", got)
	}

	if checks := New(f.URL, "wrong", state).Doctor(context.Background()); len(checks) != 1 || checks[0].Err == nil {
		t.Errorf("bad key: %+v", checks)
	}
	if checks := New(newFake(t, map[string]string{}).URL, "lin_api_1", state).Doctor(context.Background()); len(checks) != 3 || checks[1].Err == nil {
		t.Errorf("notifications failure not reported: %+v", checks)
	}
}

func TestCutBodyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte(`{"data":`))
	}))
	defer srv.Close()
	if events := New(srv.URL, "lin_api_1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("got %v", events)
	}
}
