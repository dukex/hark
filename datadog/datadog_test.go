package datadog

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/dukex/hark"
)

var now = time.Date(2026, 11, 23, 21, 0, 0, 0, time.UTC)

const app = "https://app.datadoghq.com"

type fakeDatadog struct {
	*httptest.Server
	mu      sync.Mutex
	pages   map[string]string
	queries []url.Values
}

func newFake(t *testing.T, pages map[string]string) *fakeDatadog {
	f := &fakeDatadog{pages: pages}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("DD-API-KEY") != "key" || r.Header.Get("DD-APPLICATION-KEY") != "app" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/api/v1/validate":
			_, _ = w.Write([]byte(`{"valid":true}`))
		case "/api/v2/events":
			f.queries = append(f.queries, r.URL.Query())
			if strings.Contains(r.URL.Query().Get("filter[query]"), "bad:") {
				http.Error(w, `{"errors":["invalid query"]}`, http.StatusBadRequest)
				return
			}
			body, ok := f.pages[r.URL.Query().Get("page[cursor]")]
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

func (f *fakeDatadog) watcher(t *testing.T) *Watcher {
	w := New(f.URL, app, "key", "app", filepath.Join(t.TempDir(), "datadog.json"))
	w.Now = func() time.Time { return now }
	w.state.Cursor = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	return w
}

func (f *fakeDatadog) asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.queries)
}

func (f *fakeDatadog) query(i int) url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries[i]
}

func eventJSON(id, at string, monitor int, from, to, title string) string {
	return fmt.Sprintf(`{"id":%q,"type":"event","attributes":{"timestamp":"2026-11-23T%s.123Z","message":"m","tags":["team:a"],
		"attributes":{"title":%q,"status":"error","timestamp":1,"monitor_id":%d,
		"monitor":{"id":%d,"name":"n","transition":{"transition_type":"t","source_state":%q,"destination_state":%q}}}}}`,
		id, at, title, monitor, monitor, from, to)
}

func page(after string, events ...string) string {
	return `{"data":[` + strings.Join(events, ",") + `],"meta":{"page":{"after":"` + after + `"}}}`
}

func ms(t time.Time) string { return strconv.FormatInt(t.UnixMilli(), 10) }

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

func TestMonitorStateChanges(t *testing.T) {
	f := newFake(t, map[string]string{
		"": page("c2",
			eventJSON("e1", "20:10:00", 11, "OK", "Alert", "[Triggered] CPU high on web"),
			eventJSON("e2", "20:20:00", 11, "Alert", "Alert", "[Re-notify] CPU high on web")),
		"c2": page("",
			eventJSON("e3", "20:30:00", 11, "Alert", "Warn", "[Warn] CPU high on web"),
			eventJSON("e4", "20:40:00", 12, "OK", "No Data", "[No Data] Queue depth"),
			eventJSON("e5", "20:50:00", 11, "Warn", "OK", "[Recovered] CPU high on web")),
	})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:10:00Z datadog datadog.alert "[Triggered] CPU high on web" `+app+`/monitors/11`,
		`2026-11-23T20:30:00Z datadog datadog.warn "[Warn] CPU high on web" `+app+`/monitors/11`,
		`2026-11-23T20:40:00Z datadog datadog.no_data "[No Data] Queue depth" `+app+`/monitors/12`,
		`2026-11-23T20:50:00Z datadog datadog.recovered "[Recovered] CPU high on web" `+app+`/monitors/11`)

	q := f.queries[0]
	if q.Get("filter[query]") != "source:alert" || q.Get("filter[from]") != ms(time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)) || q.Get("filter[to]") != ms(now) ||
		q.Get("sort") != "timestamp" || q.Get("page[limit]") != "1000" || f.queries[1].Get("page[cursor]") != "c2" {
		t.Errorf("queries %v", f.queries)
	}
}

func TestOverlapIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		eventJSON("e0", "20:10:00", 1, "OK", "Alert", "old"),
		eventJSON("e1", "20:58:00", 1, "Alert", "OK", "late"))})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:10:00Z datadog datadog.alert "old" `+app+`/monitors/1`,
		`2026-11-23T20:58:00Z datadog datadog.recovered "late" `+app+`/monitors/1`)
	if !w.state.Cursor.Equal(now.Add(-lag)) || strings.Join(w.state.Seen, ",") != "e1" {
		t.Errorf("state %+v", w.state)
	}
	assertLines(t, w.poll(context.Background()))
	if got := f.queries[1].Get("filter[from]"); got != ms(now.Add(-lag)) {
		t.Errorf("second poll from %s", got)
	}

	w.state.Cursor = now.Add(-time.Minute)
	w.poll(context.Background())
	if !w.state.Cursor.Equal(now.Add(-time.Minute)) {
		t.Errorf("cursor moved back to %v", w.state.Cursor)
	}
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	for name, pages := range map[string]map[string]string{
		"events unavailable":  {},
		"second page missing": {"": page("gone")},
		"bad json":            {"": "nope"},
		"bad timestamp":       {"": `{"data":[{"id":"e","attributes":{"timestamp":"yesterday"}}]}`},
	} {
		t.Run(name, func(t *testing.T) {
			w := newFake(t, pages).watcher(t)
			before := w.state.Cursor
			events := w.poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" || events[0].Provider != "datadog" {
				t.Fatalf("want one error event, got %v", events)
			}
			if !w.state.Cursor.Equal(before) {
				t.Errorf("cursor moved after a failure")
			}
		})
	}
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		if events := New(api, app, "key", "app", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesAndSavesState(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", eventJSON("e1", "20:10:00", 1, "OK", "Alert", "down"))})
	state := filepath.Join(t.TempDir(), "datadog.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:00:00Z"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, app, "key", "app", state)
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	if err := w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() }); err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z datadog datadog.alert "down" `+app+`/monitors/1`)
	if from := f.query(0).Get("filter[from]"); from != ms(time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)) {
		t.Errorf("from %s", from)
	}
	var saved struct {
		Cursor time.Time `json:"cursor"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Cursor.Equal(now.Add(-lag)) {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackAndPollsAgain(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	w := New(f.URL, app, "key", "app", filepath.Join(t.TempDir(), "datadog.json"))
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
	if got := f.query(0).Get("filter[from]"); got != ms(now.Add(-time.Hour)) {
		t.Errorf("from %s", got)
	}
}

func TestSinceOverridesTheSavedCursor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	state := filepath.Join(t.TempDir(), "datadog.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"e1"}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, app, "key", "app", state)
	w.Now = func() time.Time { return now }
	w.Since = now.Add(-3 * time.Hour)
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
	if got := f.query(0).Get("filter[from]"); got != ms(w.Since) {
		t.Errorf("from %s", got)
	}
}

func TestFilterNarrowsTheQueryAndIsCheckedOnStart(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	w := f.watcher(t)
	w.Filter = "service:api"
	w.poll(context.Background())
	if got := f.queries[0].Get("filter[query]"); got != "source:alert (service:api)" {
		t.Errorf("query %q", got)
	}

	w = New(f.URL, app, "key", "app", filepath.Join(t.TempDir(), "datadog.json"))
	w.Filter = "bad:"
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != `--filter "bad:": GET /api/v2/events: 400 Bad Request` {
		t.Fatalf("err %v", err)
	}

	f = newFake(t, map[string]string{"": page("")})
	w = New(f.URL, app, "key", "app", filepath.Join(t.TempDir(), "datadog.json"))
	w.Filter = "service:api"
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
	if probe := f.query(0); probe.Get("filter[query]") != "source:alert (service:api)" || probe.Get("page[limit]") != "1" {
		t.Errorf("probe %v", probe)
	}
}

func TestWatchFailsOnUnreadableState(t *testing.T) {
	if err := New(newFake(t, nil).URL, app, "key", "app", t.TempDir()).Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")})
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, app, "key", "app", filepath.Join(dir, "datadog.json"))
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
	state := filepath.Join(t.TempDir(), "datadog.json")
	var got []string
	for _, c := range New(f.URL, app, "key", "app", state).Doctor(context.Background()) {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "|") != "api key: valid|events: monitor events are readable|state: "+state {
		t.Errorf("got %v", got)
	}
	if q := f.queries[0]; q.Get("filter[query]") != "source:alert" || q.Get("page[limit]") != "1" {
		t.Errorf("events query %v", q)
	}

	if checks := New(f.URL, app, "wrong", "app", state).Doctor(context.Background()); len(checks) != 1 || checks[0].Err == nil {
		t.Errorf("bad api key: %+v", checks)
	}
	noEvents := newFake(t, map[string]string{})
	if checks := New(noEvents.URL, app, "key", "app", state).Doctor(context.Background()); len(checks) != 3 || checks[1].Err == nil {
		t.Errorf("events failure not reported: %+v", checks)
	}
}
