package slack

import (
	"cmp"
	"context"
	"fmt"
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

type fakeSlack struct {
	*httptest.Server
	mu      sync.Mutex
	pages   map[string]string
	auth    string
	queries []url.Values
}

func newFake(t *testing.T, pages map[string]string) *fakeSlack {
	f := &fakeSlack{pages: pages, auth: `{"ok":true,"user_id":"U1","user":"duke","team":"Acme","url":"https://example.slack.com/"}`}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer xoxp-1" {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
			return
		}
		switch r.URL.Path {
		case "/auth.test":
			_, _ = w.Write([]byte(f.auth))
		case "/search.messages":
			f.queries = append(f.queries, r.URL.Query())
			first, _, _ := strings.Cut(r.URL.Query().Get("query"), " ")
			body, ok := f.pages[first+"|"+cmp.Or(r.URL.Query().Get("page"), "1")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(body))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeSlack) watcher(t *testing.T) *Watcher {
	w := New(f.URL, "xoxp-1", filepath.Join(t.TempDir(), "slack.json"))
	w.Now = func() time.Time { return now }
	w.me = "U1"
	w.state.Cursor = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	return w
}

func (f *fakeSlack) asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queries)
}

func (f *fakeSlack) query(i int) url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries[i]
}

func ts(clock string) string {
	t, _ := time.Parse(time.DateTime, "2026-11-23 "+clock)
	return fmt.Sprintf("%d.000100", t.Unix())
}

func link(channel, clock string) string {
	return "https://example.slack.com/archives/" + channel + "/p" + strings.ReplaceAll(ts(clock), ".", "")
}

func channel(id, name string) string {
	return fmt.Sprintf(`{"id":%q,"name":%q,"is_mpim":false,"is_private":false}`, id, name)
}

func dm(id string) string {
	return fmt.Sprintf(`{"id":%q,"name":"U9","is_mpim":false,"is_private":true}`, id)
}

func groupDM(id string) string {
	return fmt.Sprintf(`{"id":%q,"name":"mpdm-a--b-1","is_mpim":true,"is_private":true}`, id)
}

func msg(kind, ch, chID, user, username, clock, text string) string {
	return fmt.Sprintf(`{"type":%q,"channel":%s,"user":%q,"username":%q,"ts":%q,"text":%q,"permalink":%q}`,
		kind, ch, user, username, ts(clock), text, link(chID, clock))
}

func page(pages int, matches ...string) string {
	return fmt.Sprintf(`{"ok":true,"messages":{"total":%d,"matches":[%s],"paging":{"pages":%d}}}`, len(matches), strings.Join(matches, ","), pages)
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

func TestMentionsAndDMs(t *testing.T) {
	f := newFake(t, map[string]string{
		"<@U1>|1": page(2,
			msg("im", dm("D4"), "D4", "U5", "erin", "20:50:00", "<@U1> ping"),
			msg("message", channel("C1", "deploys"), "C1", "U2", "ana", "20:30:00", "<@U1> can you look at &lt;this&gt; &amp; that?"),
			msg("message", channel("C1", "deploys"), "C1", "U1", "duke", "20:25:00", "<@U1> note to self")),
		"<@U1>|2": page(2,
			msg("message", channel("C2", "general"), "C2", "U3", "carol", "20:10:00", "<@U1> deploy?"),
			msg("message", channel("C2", "general"), "C2", "U3", "carol", "20:05:00", "<@U1|duke> named")),
		"to:me|1": page(5,
			msg("im", dm("D4"), "D4", "U5", "erin", "20:50:00", "<@U1> ping"),
			msg("message", groupDM("G1"), "G1", "U4", "dan", "20:45:00", "cc <@U1|duke>"),
			msg("message", groupDM("G1"), "G1", "U4", "dan", "20:40:00", "standup moved"),
			msg("message", channel("C3", "random"), "C3", "U6", "fay", "20:35:00", "not for me"),
			msg("im", dm("D2"), "D2", "U7", "bob", "20:20:00", "lunch?"),
			msg("im", dm("D2"), "D2", "U7", "bob", "19:00:00", "before the cursor")),
	})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z slack slack.mentioned #general @carol "<@U1|duke> named" `+link("C2", "20:05:00"),
		`2026-11-23T20:10:00Z slack slack.mentioned #general @carol "<@U1> deploy?" `+link("C2", "20:10:00"),
		`2026-11-23T20:20:00Z slack slack.dm @bob "lunch?" `+link("D2", "20:20:00"),
		`2026-11-23T20:30:00Z slack slack.mentioned #deploys @ana "<@U1> can you look at <this> & that?" `+link("C1", "20:30:00"),
		`2026-11-23T20:40:00Z slack slack.dm @dan "standup moved" `+link("G1", "20:40:00"),
		`2026-11-23T20:45:00Z slack slack.mentioned @dan "cc <@U1|duke>" `+link("G1", "20:45:00"),
		`2026-11-23T20:50:00Z slack slack.mentioned @erin "<@U1> ping" `+link("D4", "20:50:00"))

	var got []string
	for _, q := range f.queries {
		got = append(got, q.Get("query")+" page "+q.Get("page"))
		if q.Get("sort") != "timestamp" || q.Get("sort_dir") != "desc" || q.Get("count") != "100" {
			t.Errorf("query %v", q)
		}
	}
	if want := "<@U1> after:2026-11-21 page 1|<@U1> after:2026-11-21 page 2|to:me after:2026-11-21 page 1"; strings.Join(got, "|") != want {
		t.Errorf("queries %v", got)
	}
}

func TestOverlapIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{
		"<@U1>|1": page(1,
			msg("message", channel("C1", "deploys"), "C1", "U2", "ana", "20:58:00", "<@U1> late"),
			msg("message", channel("C1", "deploys"), "C1", "U2", "ana", "20:10:00", "<@U1> old")),
		"to:me|1": page(1),
	})
	w := f.watcher(t)

	if events := w.poll(context.Background()); len(events) != 2 {
		t.Fatalf("got %v", events)
	}
	if !w.state.Cursor.Equal(now.Add(-lag)) || strings.Join(w.state.Seen, ",") != "C1:"+ts("20:58:00") {
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
		"search unavailable":  {},
		"second page missing": {"<@U1>|1": page(2)},
		"dm search missing":   {"<@U1>|1": page(1)},
		"not ok":              {"<@U1>|1": `{"ok":false,"error":"missing_scope"}`},
		"bad json":            {"<@U1>|1": "nope"},
		"bad ts":              {"<@U1>|1": `{"ok":true,"messages":{"matches":[{"ts":"yesterday"}],"paging":{"pages":1}}}`},
	} {
		t.Run(name, func(t *testing.T) {
			w := newFake(t, pages).watcher(t)
			before := w.state.Cursor
			events := w.poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" || events[0].Provider != "slack" {
				t.Fatalf("want one error event, got %v", events)
			}
			if !w.state.Cursor.Equal(before) {
				t.Error("cursor moved after a failure")
			}
		})
	}
	if events := newFake(t, map[string]string{"<@U1>|1": `{"ok":false,"error":"missing_scope"}`}).watcher(t).poll(context.Background()); events[0].Text != "search.messages: missing_scope" {
		t.Errorf("text %q", events[0].Text)
	}
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		if events := New(api, "xoxp-1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesAndSavesState(t *testing.T) {
	f := newFake(t, map[string]string{
		"<@U1>|1": page(1, msg("message", channel("C1", "deploys"), "C1", "U2", "ana", "20:10:00", "<@U1> oi")),
		"to:me|1": page(1),
	})
	state := filepath.Join(t.TempDir(), "slack.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL+"/", "xoxp-1", state)
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	if err := w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() }); err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z slack slack.mentioned #deploys @ana "<@U1> oi" `+link("C1", "20:10:00"))
	var saved struct {
		Cursor time.Time `json:"cursor"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Cursor.Equal(now.Add(-lag)) {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackAndPollsAgain(t *testing.T) {
	f := newFake(t, map[string]string{"<@U1>|1": page(1), "to:me|1": page(1)})
	w := New(f.URL, "xoxp-1", filepath.Join(t.TempDir(), "slack.json"))
	w.Now = func() time.Time { return now }
	w.Interval = time.Millisecond
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	go func() {
		for f.asked() < 4 {
			time.Sleep(time.Millisecond)
		}
		cancel()
	}()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if got := f.query(0).Get("query"); got != "<@U1> after:2026-11-21" {
		t.Errorf("query %q", got)
	}
}

func TestSinceOverridesTheSavedCursor(t *testing.T) {
	f := newFake(t, map[string]string{"<@U1>|1": page(1), "to:me|1": page(1)})
	state := filepath.Join(t.TempDir(), "slack.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"C1:1"}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "xoxp-1", state)
	w.Now = func() time.Time { return now }
	w.Since = time.Date(2026, 11, 10, 9, 0, 0, 0, time.UTC)
	w.Interval = time.Millisecond
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
	if got := f.query(0).Get("query"); got != "<@U1> after:2026-11-08" {
		t.Errorf("query %q", got)
	}
}

func TestFilterNarrowsTheSearchAndIsCheckedOnStart(t *testing.T) {
	f := newFake(t, map[string]string{"<@U1>|1": page(1), "to:me|1": page(1)})
	w := f.watcher(t)
	w.Filter = "-in:#random"
	w.poll(context.Background())
	if a, b := f.queries[0].Get("query"), f.queries[1].Get("query"); a != "<@U1> after:2026-11-21 -in:#random" || b != "to:me after:2026-11-21 -in:#random" {
		t.Errorf("queries %q %q", a, b)
	}

	w = New(newFake(t, map[string]string{}).URL, "xoxp-1", filepath.Join(t.TempDir(), "slack.json"))
	w.Filter = "in:#gone"
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != `--filter "in:#gone": GET /search.messages: 404 Not Found` {
		t.Fatalf("err %v", err)
	}

	f = newFake(t, map[string]string{"<@U1>|1": page(1), "to:me|1": page(1)})
	w = New(f.URL, "xoxp-1", filepath.Join(t.TempDir(), "slack.json"))
	w.Filter = "in:#deploys"
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
	if probe := f.query(0); probe.Get("query") != "<@U1> in:#deploys" || probe.Get("count") != "1" {
		t.Errorf("probe %v", probe)
	}
}

func TestWatchFailsWithoutLoginOrState(t *testing.T) {
	f := newFake(t, nil)
	if err := New(f.URL, "wrong", "").Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != "auth.test: invalid_auth" {
		t.Errorf("login: %v", err)
	}
	if err := New(f.URL, "xoxp-1", t.TempDir()).Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"<@U1>|1": page(1), "to:me|1": page(1)})
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "xoxp-1", filepath.Join(dir, "slack.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	_ = w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() })
	if len(got) != 1 || got[0].Kind != "error" {
		t.Fatalf("got %v", got)
	}
}

func TestDoctor(t *testing.T) {
	f := newFake(t, map[string]string{
		"<@U1>|1": page(1, msg("message", channel("C1", "deploys"), "C1", "U2", "ana", "20:10:00", "<@U1> oi")),
		"to:me|1": page(1),
	})
	state := filepath.Join(t.TempDir(), "slack.json")
	w := New(f.URL, "xoxp-1", state)
	w.Now = func() time.Time { return now }
	var got []string
	for _, c := range w.Doctor(context.Background()) {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "|") != "token: logged in as duke on Acme|search: 1 mentions and 0 DMs in the last 7 days|state: "+state {
		t.Errorf("got %v", got)
	}
	if q := f.queries[0]; q.Get("query") != "<@U1> after:2026-11-15" || q.Get("count") != "1" {
		t.Errorf("search %v", q)
	}

	if checks := New(f.URL, "wrong", state).Doctor(context.Background()); len(checks) != 1 || checks[0].Err == nil {
		t.Errorf("bad token: %+v", checks)
	}
	for _, pages := range []map[string]string{{}, {"<@U1>|1": page(1)}} {
		if checks := New(newFake(t, pages).URL, "xoxp-1", state).Doctor(context.Background()); len(checks) != 3 || checks[1].Err == nil {
			t.Errorf("search failure not reported: %+v", checks)
		}
	}
}

func TestCutBodyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte(`{"ok":`))
	}))
	defer srv.Close()
	if events := New(srv.URL, "xoxp-1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("got %v", events)
	}
}
