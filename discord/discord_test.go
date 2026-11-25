package discord

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

var (
	now   = time.Date(2026, 11, 23, 21, 0, 0, 0, time.UTC)
	start = time.Date(2026, 11, 23, 20, 0, 0, 0, time.UTC)
)

type fakeDiscord struct {
	*httptest.Server
	mu       sync.Mutex
	pages    map[string]string
	guilds   string
	flags    int
	searches []url.Values
	paths    []string
}

func newFake(t *testing.T, pages map[string]string) *fakeDiscord {
	f := &fakeDiscord{pages: pages, guilds: `[{"id":"G1","name":"Acme Dev"}]`, flags: 1 << 19}
	f.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "DiscordBot (https://github.com/dukex/hark, ") {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`error code: 1010`))
			return
		}
		if r.Header.Get("Authorization") != "Bot tok" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"message":"401: Unauthorized","code":0}`))
			return
		}
		switch r.URL.Path {
		case "/users/@me":
			_, _ = w.Write([]byte(`{"id":"B1","username":"harkbot"}`))
			return
		case "/users/U1":
			_, _ = w.Write([]byte(`{"id":"U1","username":"duke"}`))
			return
		case "/users/@me/guilds":
			_, _ = w.Write([]byte(f.guilds))
			return
		case "/applications/@me":
			_, _ = fmt.Fprintf(w, `{"id":"A1","flags":%d}`, f.flags)
			return
		}
		guild, ok := strings.CutSuffix(strings.TrimPrefix(r.URL.Path, "/guilds/"), "/messages/search")
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"Unknown User","code":10013}`))
			return
		}
		q := r.URL.Query()
		f.searches = append(f.searches, q)
		f.paths = append(f.paths, guild)
		search := "mentions"
		if q.Has("replied_to_user_id") {
			search = "replies"
		}
		body, ok := f.pages[guild+"|"+search+"|"+cmp.Or(q.Get("offset"), "0")]
		switch {
		case q.Get("author_type") == "bad":
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(`{"message":"Invalid Form Body","code":50035}`))
		case body == "202":
			w.WriteHeader(http.StatusAccepted)
			_, _ = w.Write([]byte(`{"message":"Index not yet available. Try again later","code":110000,"documents_indexed":0,"retry_after":2}`))
		case !ok:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"Missing Access","code":50001}`))
		default:
			_, _ = w.Write([]byte(body))
		}
	}))
	t.Cleanup(f.Close)
	return f
}

func (f *fakeDiscord) watcher(t *testing.T) *Watcher {
	w := New(f.URL, "tok", "U1", filepath.Join(t.TempDir(), "discord.json"))
	w.Now = func() time.Time { return now }
	w.state.Guilds = map[string]cursor{"G1": {Cursor: start}, "G2": {Cursor: start}}
	return w
}

func (f *fakeDiscord) asked() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.searches)
}

func (f *fakeDiscord) search(i int) url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.searches[i]
}

func link(guild, channel, id string) string {
	return "https://discord.com/channels/" + guild + "/" + channel + "/" + id
}

func msg(id, channel, author, username, clock, content string) string {
	return fmt.Sprintf(`[{"id":%q,"channel_id":%q,"author":{"id":%q,"username":%q},"content":%q,"timestamp":"2026-11-23T%s.123000+00:00"}]`,
		id, channel, author, username, content, clock)
}

func hits(total int, messages ...string) string {
	return fmt.Sprintf(`{"total_results":%d,"messages":[%s]}`, total, strings.Join(messages, ","))
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

func TestMentionsAndReplies(t *testing.T) {
	f := newFake(t, map[string]string{
		"G1|mentions|0": hits(27,
			msg("m5", "C1", "U2", "ana", "20:50:00", "<@U1> can you look?"),
			msg("m4", "C1", "U1", "duke", "20:45:00", "<@U1> note to self")),
		"G1|mentions|25": hits(27,
			msg("m3", "C2", "U3", "bob", "20:30:00", "ping <@U1>")),
		"G1|replies|0": hits(2,
			msg("m5", "C1", "U2", "ana", "20:50:00", "<@U1> can you look?"),
			msg("m2", "C2", "U4", "carol", "20:20:00", "agreed, shipping it")),
	})
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:20:00Z discord discord.replied @carol "Acme Dev: agreed, shipping it" `+link("G1", "C2", "m2"),
		`2026-11-23T20:30:00Z discord discord.mentioned @bob "Acme Dev: ping <@U1>" `+link("G1", "C2", "m3"),
		`2026-11-23T20:50:00Z discord discord.mentioned @ana "Acme Dev: <@U1> can you look?" `+link("G1", "C1", "m5"))

	var got []string
	for i, q := range f.searches {
		got = append(got, f.paths[i]+" "+q.Get("mentions")+q.Get("replied_to_user_id")+" "+q.Get("offset"))
		if q.Get("min_id") != "1574514878054400000" || q.Get("sort_by") != "timestamp" || q.Get("sort_order") != "desc" || q.Get("limit") != "25" {
			t.Errorf("search %v", q)
		}
	}
	if strings.Join(got, "|") != "G1 U1 0|G1 U1 25|G1 U1 0" {
		t.Errorf("searches %v", got)
	}
}

func TestOverlapIsReportedOnce(t *testing.T) {
	f := newFake(t, map[string]string{
		"G1|mentions|0": hits(2,
			msg("m2", "C1", "U2", "ana", "20:58:00", "<@U1> late"),
			msg("m1", "C1", "U2", "ana", "19:10:00", "<@U1> before the cursor")),
		"G1|replies|0": hits(0),
	})
	w := f.watcher(t)

	if events := w.poll(context.Background()); len(events) != 1 {
		t.Fatalf("got %v", events)
	}
	if g := w.state.Guilds["G1"]; !g.Cursor.Equal(now.Add(-lag)) || strings.Join(g.Seen, ",") != "m2" {
		t.Errorf("state %+v", g)
	}
	assertLines(t, w.poll(context.Background()))

	w.state.Guilds["G1"] = cursor{Cursor: now.Add(-time.Minute)}
	w.poll(context.Background())
	if g := w.state.Guilds["G1"]; !g.Cursor.Equal(now.Add(-time.Minute)) {
		t.Errorf("cursor moved back to %v", g.Cursor)
	}
}

func TestAFailingServerDoesNotHoldBackTheOthers(t *testing.T) {
	f := newFake(t, map[string]string{
		"G2|mentions|0": hits(1, msg("m1", "C9", "U2", "ana", "20:30:00", "<@U1> hi")),
		"G2|replies|0":  hits(0),
		"G3|mentions|0": "202",
	})
	f.guilds = `[{"id":"G1","name":"Acme Dev"},{"id":"G2","name":"Friends"},{"id":"G3","name":"New"}]`
	w := f.watcher(t)

	assertLines(t, w.poll(context.Background()),
		`2026-11-23T20:30:00Z discord discord.mentioned @ana "Friends: <@U1> hi" `+link("G2", "C9", "m1"),
		`2026-11-23T21:00:00Z discord error "Acme Dev: GET /guilds/G1/messages/search: 403 Forbidden: Missing Access"`,
		`2026-11-23T21:00:00Z discord error "New: GET /guilds/G3/messages/search: Index not yet available. Try again later"`)
	if !w.state.Guilds["G1"].Cursor.Equal(start) || !w.state.Guilds["G2"].Cursor.Equal(now.Add(-lag)) {
		t.Errorf("state %+v", w.state)
	}
	if _, ok := w.state.Guilds["G3"]; ok {
		t.Error("a server that failed got a cursor")
	}
}

func TestFailuresBecomeErrorEvents(t *testing.T) {
	for name, pages := range map[string]map[string]string{
		"second page missing":  {"G1|mentions|0": hits(30)},
		"reply search missing": {"G1|mentions|0": hits(0)},
		"bad json":             {"G1|mentions|0": "nope"},
	} {
		t.Run(name, func(t *testing.T) {
			w := newFake(t, pages).watcher(t)
			events := w.poll(context.Background())
			if len(events) != 1 || events[0].Kind != "error" || events[0].Provider != "discord" {
				t.Fatalf("want one error event, got %v", events)
			}
			if !w.state.Guilds["G1"].Cursor.Equal(start) {
				t.Error("cursor moved after a failure")
			}
		})
	}
	f := newFake(t, nil)
	f.guilds = "nope"
	if events := f.watcher(t).poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("server list: got %v", events)
	}
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "oops", http.StatusInternalServerError)
	}))
	defer plain.Close()
	if events := New(plain.URL, "tok", "U1", "").poll(context.Background()); len(events) != 1 || events[0].Text != "GET /users/@me/guilds: 500 Internal Server Error" {
		t.Fatalf("plain error: got %v", events)
	}
	for _, api := range []string{"http://127.0.0.1:1", "http://bad host"} {
		if events := New(api, "tok", "U1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
			t.Fatalf("%s: got %v", api, events)
		}
	}
}

func TestWatchResumesAndSavesState(t *testing.T) {
	f := newFake(t, map[string]string{
		"G1|mentions|0": hits(1, msg("m1", "C1", "U2", "ana", "20:10:00", "<@U1> oi")),
		"G1|replies|0":  hits(0),
	})
	state := filepath.Join(t.TempDir(), "discord.json")
	if err := hark.SaveState(state, map[string]any{"guilds": map[string]any{"G1": map[string]any{"cursor": "2026-11-23T20:00:00Z"}}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL+"/", "tok", "U1", state)
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	if err := w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() }); err != nil {
		t.Fatal(err)
	}
	assertLines(t, got, `2026-11-23T20:10:00Z discord discord.mentioned @ana "Acme Dev: <@U1> oi" `+link("G1", "C1", "m1"))
	var saved struct {
		Guilds map[string]cursor `json:"guilds"`
	}
	if err := hark.LoadState(state, &saved); err != nil || !saved.Guilds["G1"].Cursor.Equal(now.Add(-lag)) {
		t.Errorf("saved %+v, %v", saved, err)
	}
}

func TestWatchStartsOneHourBackAndPollsAgain(t *testing.T) {
	f := newFake(t, map[string]string{"G1|mentions|0": hits(0), "G1|replies|0": hits(0)})
	w := New(f.URL, "tok", "U1", filepath.Join(t.TempDir(), "discord.json"))
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
	if got := f.search(0).Get("min_id"); got != "1574514878054400000" {
		t.Errorf("min_id %s", got)
	}
}

func TestSinceOverridesTheSavedCursors(t *testing.T) {
	f := newFake(t, map[string]string{"G1|mentions|0": hits(0), "G1|replies|0": hits(0)})
	state := filepath.Join(t.TempDir(), "discord.json")
	if err := hark.SaveState(state, map[string]any{"guilds": map[string]any{"G1": map[string]any{"cursor": "2026-11-23T20:50:00Z", "seen": []string{"m1"}}}}); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "tok", "U1", state)
	w.Now = func() time.Time { return now }
	w.Since = start
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
	if got := f.search(0).Get("min_id"); got != "1574514878054400000" {
		t.Errorf("min_id %s", got)
	}
}

func TestFilterNarrowsTheSearchAndIsCheckedOnStart(t *testing.T) {
	f := newFake(t, map[string]string{"G1|mentions|0": hits(0), "G1|replies|0": hits(0)})
	w := f.watcher(t)
	w.Filter = "channel_id=C1&channel_id=C2&author_type=-bot"
	w.poll(context.Background())
	for _, q := range f.searches {
		if strings.Join(q["channel_id"], ",") != "C1,C2" || q.Get("author_type") != "-bot" {
			t.Errorf("search %v", q)
		}
	}

	for filter, want := range map[string]string{
		"author_type=bad": `--filter "author_type=bad": GET /guilds/G1/messages/search: 400 Bad Request: Invalid Form Body`,
		"channel_id=%zz":  `--filter "channel_id=%zz": invalid URL escape "%zz"`,
	} {
		w = New(newFake(t, nil).URL, "tok", "U1", filepath.Join(t.TempDir(), "discord.json"))
		w.Filter = filter
		if err := w.Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != want {
			t.Errorf("%s: err %v", filter, err)
		}
	}

	f = newFake(t, map[string]string{"G1|mentions|0": hits(0), "G1|replies|0": hits(0)})
	w = New(f.URL, "tok", "U1", filepath.Join(t.TempDir(), "discord.json"))
	w.Filter = "author_type=user"
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
	if probe := f.search(0); probe.Get("limit") != "1" || probe.Get("author_type") != "user" || probe.Get("mentions") != "U1" {
		t.Errorf("probe %v", probe)
	}
}

func TestWatchChecksTokenUserAndServers(t *testing.T) {
	f := newFake(t, nil)
	if err := New(f.URL, "wrong", "U1", "").Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != "GET /users/@me: 401 Unauthorized: 401: Unauthorized" {
		t.Errorf("token: %v", err)
	}
	if err := New(f.URL, "tok", "U9", "").Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != "DISCORD_USER_ID U9: GET /users/U9: 404 Not Found: Unknown User" {
		t.Errorf("user: %v", err)
	}
	f.guilds = "[]"
	if err := New(f.URL, "tok", "U1", "").Watch(context.Background(), func(hark.Event) {}); err == nil || err.Error() != "the bot is in no server: add it to the servers to watch" {
		t.Errorf("servers: %v", err)
	}
	f.guilds = "nope"
	if err := New(f.URL, "tok", "U1", "").Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error on a bad server list")
	}
	f.guilds = `[{"id":"G1","name":"Acme Dev"}]`
	if err := New(f.URL, "tok", "U1", t.TempDir()).Watch(context.Background(), func(hark.Event) {}); err == nil {
		t.Error("want error reading a directory as state")
	}
}

func TestStateSaveFailureBecomesErrorEvent(t *testing.T) {
	f := newFake(t, map[string]string{"G1|mentions|0": hits(0), "G1|replies|0": hits(0)})
	dir := filepath.Join(t.TempDir(), "ro")
	if err := os.Mkdir(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	w := New(f.URL, "tok", "U1", filepath.Join(dir, "discord.json"))
	w.Now = func() time.Time { return now }
	ctx, cancel := context.WithCancel(context.Background())
	var got []hark.Event
	_ = w.Watch(ctx, func(e hark.Event) { got = append(got, e); cancel() })
	if len(got) != 1 || got[0].Kind != "error" {
		t.Fatalf("got %v", got)
	}
}

func checks(w *Watcher) string {
	var got []string
	for _, c := range w.Doctor(context.Background()) {
		line := c.Name + ": " + c.Detail
		if c.Err != nil {
			line = c.Name + " ✗ " + c.Err.Error()
		}
		got = append(got, line)
	}
	return strings.Join(got, "|")
}

func TestDoctor(t *testing.T) {
	f := newFake(t, map[string]string{"G1|mentions|0": hits(3), "G2|mentions|0": hits(2)})
	f.guilds = `[{"id":"G1","name":"Acme Dev"},{"id":"G2","name":"Friends"}]`
	state := filepath.Join(t.TempDir(), "discord.json")
	w := New(f.URL, "tok", "U1", state)
	w.Now = func() time.Time { return now }
	if got := checks(w); got != "token: logged in as harkbot|message content: intent on|user: watching mentions of duke|servers: Acme Dev, Friends|search: 5 mentions in the last 7 days|state: "+state {
		t.Errorf("got %s", got)
	}
	if q := f.search(0); q.Get("mentions") != "U1" || q.Get("limit") != "1" || q.Get("min_id") != snowflake(now.AddDate(0, 0, -7)) {
		t.Errorf("search %v", q)
	}

	f.pages = map[string]string{"G1|mentions|0": hits(3)}
	if got := checks(New(f.URL, "tok", "U1", state)); !strings.Contains(got, "|search ✗ Friends: GET /guilds/G2/messages/search: 403 Forbidden: Missing Access|") {
		t.Errorf("got %s", got)
	}

	f.flags, f.guilds = 1<<18, "[]"
	if got := checks(New(f.URL, "tok", "U9", state)); got != "token: logged in as harkbot|message content: intent on|user ✗ GET /users/U9: 404 Not Found: Unknown User|servers ✗ the bot is in no server: add it to the servers to watch|search: 0 mentions in the last 7 days|state: "+state {
		t.Errorf("got %s", got)
	}

	f.flags, f.guilds = 0, "nope"
	if got := checks(New(f.URL, "tok", "U1", state)); !strings.Contains(got, "message content ✗ turn on Message Content Intent on the Bot page of the app") || !strings.Contains(got, "servers ✗ invalid character") {
		t.Errorf("got %s", got)
	}

	if got := checks(New(f.URL, "wrong", "U1", state)); got != "token ✗ GET /users/@me: 401 Unauthorized: 401: Unauthorized" {
		t.Errorf("bad token: %s", got)
	}
}

func TestCutBodyIsAnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		_, _ = w.Write([]byte(`[{"id":`))
	}))
	defer srv.Close()
	if events := New(srv.URL, "tok", "U1", "").poll(context.Background()); len(events) != 1 || events[0].Kind != "error" {
		t.Fatalf("got %v", events)
	}
}
