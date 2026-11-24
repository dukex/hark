package todoist

import (
	"context"
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

var brt = time.FixedZone("BRT", -3*3600)
var now = time.Date(2026, 11, 23, 18, 0, 0, 0, brt)

type fakeTodoist struct {
	*httptest.Server
	mu      sync.Mutex
	pages   map[string]string
	today   string
	queries []url.Values
	tasks   map[string]string
	asked   []url.Values
}

func newFake(t *testing.T, pages map[string]string, today string) *fakeTodoist {
	f := &fakeTodoist{pages: pages, today: today}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/activities":
			f.queries = append(f.queries, r.URL.Query())
			body, ok := f.pages[r.URL.Query().Get("cursor")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(body))
		case "/tasks/filter", "/tasks/completed/by_completion_date":
			q := r.URL.Query()
			if q.Get("query") == "today" && f.today != "" {
				_, _ = w.Write([]byte(f.today))
				return
			}
			f.asked = append(f.asked, q)
			body, ok := f.tasks[r.URL.Path+" "+q.Get("query")+q.Get("filter_query")+" "+q.Get("cursor")]
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

func (f *fakeTodoist) watcher(t *testing.T) *Watcher {
	w := New(f.URL, "secret", filepath.Join(t.TempDir(), "todoist.json"))
	w.Now = func() time.Time { return now }
	w.state.Cursor = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
	w.state.DueDay = "2026-11-23"
	return w
}

func act(objectType, eventType, objectID, date, extra string) string {
	return `{"id":99999999999999999999999999999999999999,"object_type":"` + objectType + `","event_type":"` + eventType +
		`","object_id":"` + objectID + `","parent_item_id":"task9","event_date":"` + date + `","initiator_id":"1001","extra_data":{` + extra + `}}`
}

func page(next string, events ...string) string {
	cursor := "null"
	if next != "" {
		cursor = `"` + next + `"`
	}
	return `{"results":[` + strings.Join(events, ",") + `],"next_cursor":` + cursor + `}`
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

func TestTaskActivity(t *testing.T) {
	f := newFake(t, map[string]string{
		"": page("p2",
			act("item", "updated", "t4", "2026-11-23T20:40:00Z", `"content":"Deploy","due_date":null,"last_due_date":"2026-11-24T02:59:59.000000Z"`),
			act("item", "updated", "t3", "2026-11-23T20:30:00Z", `"content":"Revisar PR","due_date":"2026-11-26T02:59:59.000000Z","last_due_date":"2026-11-25T02:59:59.000000Z"`),
			act("item", "updated", "t2", "2026-11-23T20:20:00Z", `"content":"Renomeada","last_content":"Antiga"`)),
		"p2": page("",
			act("note", "added", "n1", "2026-11-23T20:15:00Z", `"content":"já mandei o link"`),
			act("item", "completed", "t1", "2026-11-23T20:10:00Z", `"content":"Pagar a conta de luz"`),
			act("item", "added", "t1", "2026-11-23T20:05:00.307Z", `"content":"Pagar a conta de luz","client":"Claude"`)),
	}, "")
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z todoist task.created "Pagar a conta de luz" https://app.todoist.com/app/task/t1`,
		`2026-11-23T20:10:00Z todoist task.completed "Pagar a conta de luz" https://app.todoist.com/app/task/t1`,
		`2026-11-23T20:15:00Z todoist task.commented "já mandei o link" https://app.todoist.com/app/task/task9`,
		`2026-11-23T20:30:00Z todoist task.rescheduled "Revisar PR, due 2026-11-25" https://app.todoist.com/app/task/t3`,
		`2026-11-23T20:40:00Z todoist task.rescheduled "Deploy, due removed" https://app.todoist.com/app/task/t4`)

	q := f.queries[0]
	if q.Get("object_event_types") != `["item:added","item:completed","item:updated","note:added"]` ||
		q.Get("date_from") != "2026-11-23T20:00:00Z" || q.Get("annotate_notes") != "true" || f.queries[1].Get("cursor") != "p2" {
		t.Errorf("queries %v", f.queries)
	}
}

func TestActivityIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		act("item", "added", "t2", "2026-11-23T20:05:00Z", `"content":"B"`),
		act("item", "added", "t1", "2026-11-23T20:05:00Z", `"content":"A"`),
		act("item", "added", "t0", "2026-11-23T19:00:00Z", `"content":"antes do cursor"`))}, "")
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z todoist task.created "B" https://app.todoist.com/app/task/t2`,
		`2026-11-23T20:05:00Z todoist task.created "A" https://app.todoist.com/app/task/t1`)
	assertLines(t, w.poll(context.Background()))
	if got := f.queries[1].Get("date_from"); got != "2026-11-23T20:05:00Z" {
		t.Errorf("second poll date_from %q", got)
	}
}

func TestUnparsableDueDateIsShownAsIs(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		act("item", "updated", "t1", "2026-11-23T20:05:00Z", `"content":"X","due_date":"amanhã"`))}, "")
	assertLines(t, f.watcher(t).poll(context.Background()),
		`2026-11-23T20:05:00Z todoist task.rescheduled "X, due amanhã" https://app.todoist.com/app/task/t1`)
}

func TestTasksDueTodayOncePerDay(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[{"id":"t7","content":"Pagar boleto"},{"id":"t8","content":"Ligar"}],"next_cursor":null}`)
	w := f.watcher(t)
	w.state.DueDay = "2026-11-22"

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T21:00:00Z todoist task.due "Pagar boleto" https://app.todoist.com/app/task/t7`,
		`2026-11-23T21:00:00Z todoist task.due "Ligar" https://app.todoist.com/app/task/t8`)
	assertLines(t, w.poll(context.Background()))
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	cases := map[string]struct {
		pages  map[string]string
		today  string
		dueDay string
	}{
		"activity unavailable": {pages: map[string]string{}, dueDay: "2026-11-23"},
		"second page missing":  {pages: map[string]string{"": page("gone")}, dueDay: "2026-11-23"},
		"bad json":             {pages: map[string]string{"": "nope"}, dueDay: "2026-11-23"},
		"today unavailable":    {pages: map[string]string{"": page("")}, dueDay: "2026-11-22"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			w := newFake(t, c.pages, c.today).watcher(t)
			w.state.DueDay = c.dueDay
			events := w.poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" {
				t.Fatalf("want one error event, got %v", events)
			}
		})
	}
}

func TestUnreachableAPI(t *testing.T) {
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		w := New(api, "secret", "")
		w.state.DueDay = time.Now().Format(time.DateOnly)
		if events := w.poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesFromSavedState(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		act("item", "completed", "t1", "2026-11-23T20:10:00Z", `"content":"A"`))}, `{"results":[]}`)
	state := filepath.Join(t.TempDir(), "todoist.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:00:00Z", "due_day": "2026-11-23"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL+"/", "secret", state)
	w.Now = func() time.Time { return now }
	w.Interval = time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	err := w.Watch(ctx, func(e hark.Event) {
		got = append(got, e)
		cancel()
	})
	if err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z todoist task.completed "A" https://app.todoist.com/app/task/t1`)

	var saved struct {
		Cursor time.Time `json:"cursor"`
		Seen   []string  `json:"seen"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Cursor.Equal(time.Date(2026, 11, 23, 20, 10, 0, 0, time.UTC)) || len(saved.Seen) != 1 {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackWithoutState(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[]}`)
	w := New(f.URL, "secret", filepath.Join(t.TempDir(), "todoist.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if want := now.Add(-time.Hour); !w.state.Cursor.Equal(want) {
		t.Errorf("cursor %v, want %v", w.state.Cursor, want)
	}
}

func TestWatchPollsAgainAfterInterval(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[]}`)
	w := New(f.URL, "secret", filepath.Join(t.TempDir(), "todoist.json"))
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
}

func TestWatchFailsOnCorruptState(t *testing.T) {
	dir := t.TempDir()
	w := New("http://unused", "secret", dir)
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Fatal("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[]}`)
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "secret", filepath.Join(dir, "todoist.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	_ = w.Watch(ctx, func(e hark.Event) {
		got = append(got, e)
		cancel()
	})
	if len(got) != 1 || got[0].Kind != "error" {
		t.Fatalf("got %v", got)
	}
}

func TestDoctor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[]}`)
	state := filepath.Join(t.TempDir(), "todoist.json")
	w := New(f.URL, "secret", state)
	w.Now = func() time.Time { return now }
	checks := w.Doctor(context.Background())
	var got []string
	for _, c := range checks {
		if c.Err != nil {
			t.Errorf("%s failed: %v", c.Name, c.Err)
		}
		got = append(got, c.Name+": "+c.Detail)
	}
	if strings.Join(got, "|") != "token: valid|activity log: readable|state: "+state {
		t.Errorf("got %v", got)
	}
	if q := f.queries[len(f.queries)-1]; q.Get("date_from") != "2026-11-23T21:00:00Z" || q.Get("limit") != "1" {
		t.Errorf("activity check should only look from now on: %v", q)
	}

	if checks := New(f.URL, "wrong", state).Doctor(context.Background()); len(checks) != 1 || checks[0].Err == nil {
		t.Errorf("bad token: %+v", checks)
	}
	noActivity := newFake(t, map[string]string{}, `{"results":[]}`)
	if checks := New(noActivity.URL, "secret", state).Doctor(context.Background()); len(checks) != 3 || checks[1].Err == nil {
		t.Errorf("activity log failure not reported: %+v", checks)
	}
}

func TestSinceOverridesTheSavedCursor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, `{"results":[]}`)
	state := filepath.Join(t.TempDir(), "todoist.json")
	if err := hark.SaveState(state, map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"x"}, "due_day": "2026-11-23"}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "secret", state)
	w.Now = func() time.Time { return now }
	w.Since = time.Date(2026, 11, 23, 15, 0, 0, 0, time.UTC)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := w.Watch(ctx, func(hark.Event) {}); err != nil {
		t.Fatal(err)
	}
	if !w.state.Cursor.Equal(w.Since) || w.state.Seen != nil || w.state.DueDay != "2026-11-23" {
		t.Errorf("state %+v", w.state)
	}
}

func TestFilterKeepsOnlyMatchingTasks(t *testing.T) {
	f := newFake(t, map[string]string{"": page("",
		act("item", "added", "t3", "2026-11-23T20:20:00Z", `"content":"Outro projeto"`),
		act("note", "added", "n1", "2026-11-23T20:15:00Z", `"content":"comentário"`),
		act("item", "completed", "t2", "2026-11-23T20:10:00Z", `"content":"Feita"`),
		act("item", "added", "t1", "2026-11-23T20:05:00Z", `"content":"Nova"`))}, "")
	f.tasks = map[string]string{
		"/tasks/filter #Work & p1 ":                       `{"results":[{"id":"t1"}],"next_cursor":"c2"}`,
		"/tasks/filter #Work & p1 c2":                     `{"results":[{"id":"task9"}],"next_cursor":null}`,
		"/tasks/completed/by_completion_date #Work & p1 ": `{"items":[{"id":"t2"}]}`,
	}
	w := f.watcher(t)
	w.Filter = "#Work & p1"

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:05:00Z todoist task.created "Nova" https://app.todoist.com/app/task/t1`,
		`2026-11-23T20:10:00Z todoist task.completed "Feita" https://app.todoist.com/app/task/t2`,
		`2026-11-23T20:15:00Z todoist task.commented "comentário" https://app.todoist.com/app/task/task9`)
	if done := f.asked[2]; done.Get("since") != "2026-11-23T20:05:00Z" || done.Get("until") != "2026-11-23T20:20:01Z" {
		t.Errorf("completed window %v", done)
	}
	if !w.state.Cursor.Equal(time.Date(2026, 11, 23, 20, 20, 0, 0, time.UTC)) {
		t.Errorf("filtered events should still move the cursor: %v", w.state.Cursor)
	}
}

func TestFilterFailureKeepsTheCursor(t *testing.T) {
	f := newFake(t, map[string]string{"": page("", act("item", "added", "t1", "2026-11-23T20:05:00Z", `"content":"Nova"`))}, "")
	w := f.watcher(t)
	w.Filter = "#Work"
	events := w.poll(context.Background())
	if len(events) != 1 || events[0].Kind != "error" || !strings.Contains(events[0].Text, "/tasks/filter: 404") {
		t.Fatalf("got %v", events)
	}
	if !w.state.Cursor.Equal(time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)) {
		t.Errorf("cursor moved to %v", w.state.Cursor)
	}
	f.tasks = map[string]string{"/tasks/filter #Work ": `{"results":[],"next_cursor":null}`}
	if events := w.poll(context.Background()); len(events) != 1 || !strings.Contains(events[0].Text, "/tasks/completed/by_completion_date: 404") {
		t.Fatalf("got %v", events)
	}
}

func TestFilterWithoutActivityAsksNothing(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, "")
	w := f.watcher(t)
	w.Filter = "#Work"
	if events := w.poll(context.Background()); len(events) != 0 || len(f.asked) != 0 {
		t.Errorf("events %v, asked %v", events, f.asked)
	}
}

func TestFilterNarrowsTasksDueToday(t *testing.T) {
	f := newFake(t, map[string]string{"": page("")}, "")
	f.tasks = map[string]string{"/tasks/filter today & (#Work) ": `{"results":[{"id":"t1","content":"Deploy"}],"next_cursor":null}`}
	w := f.watcher(t)
	w.state.DueDay = ""
	w.Filter = "#Work"
	assertLines(t, w.poll(context.Background()), `2026-11-23T21:00:00Z todoist task.due "Deploy" https://app.todoist.com/app/task/t1`)
}

func TestFilterIsCheckedOnStart(t *testing.T) {
	f := newFake(t, map[string]string{}, `{"results":[]}`)
	w := New(f.URL, "secret", filepath.Join(t.TempDir(), "todoist.json"))
	w.Now = func() time.Time { return now }
	w.Filter = "#Wrok"
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || !strings.Contains(err.Error(), `--filter "#Wrok": GET /tasks/filter: 404`) {
		t.Fatalf("err %v", err)
	}
	f.tasks = map[string]string{"/tasks/filter #Work ": `{"results":[],"next_cursor":null}`}
	w.Filter = "#Work"
	if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || !strings.Contains(err.Error(), `--filter "#Work": GET /tasks/completed/by_completion_date: 404`) {
		t.Fatalf("err %v", err)
	}
	f.tasks["/tasks/completed/by_completion_date #Work "] = `{"items":[]}`
	ctx, cancel := context.WithCancel(context.Background())
	if err := w.Watch(ctx, func(hark.Event) { cancel() }); err != nil {
		t.Fatal(err)
	}
	if got := f.asked[1]; got.Get("query") != "#Work" || got.Get("limit") != "1" {
		t.Errorf("probe %v", got)
	}
	if got := f.asked[len(f.asked)-2]; got.Get("filter_query") != "#Work" || got.Get("since") != "2026-11-23T20:00:00Z" || got.Get("until") != "2026-11-23T21:00:00Z" || got.Get("limit") != "1" {
		t.Errorf("completed probe %v", got)
	}
}
