package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark"
)

type polls struct {
	batches [][]hark.Event
	saved   *int
}

func (p polls) Watch(ctx context.Context, emit func(hark.Event)) error {
	for _, batch := range p.batches {
		for _, e := range batch {
			emit(e)
		}
		*p.saved++
		select {
		case <-ctx.Done():
			return nil
		default:
		}
	}
	return nil
}

func (polls) Doctor(context.Context) []hark.Check { return nil }

func TestExitOnEventStopsAfterThePollThatPrintedAnEvent(t *testing.T) {
	event := func(kind, text string) hark.Event { return hark.Event{Provider: "jira", Kind: kind, Text: text} }
	batches := [][]hark.Event{
		{event("error", "GET /rest/api/3/search/jql: 502")},
		{},
		{event("jira.commented", "Ana: ok"), event("jira.assigned", "PROJ-7")},
		{event("jira.mentioned", "later")},
	}
	for _, tc := range []struct {
		args  []string
		lines []string
		saved int
	}{
		{[]string{"jira"}, []string{"502", "Ana: ok", "PROJ-7", "later"}, 4},
		{[]string{"jira", "--exit-on-event"}, []string{"502", "Ana: ok", "PROJ-7"}, 3},
	} {
		saved := 0
		withProviders(t, map[string]constructor{"jira": provider(polls{batches, &saved}, nil)})
		var stdout, stderr bytes.Buffer
		code := run(context.Background(), tc.args, env(map[string]string{"HARK_NO_UPDATE": "1"}), &stdout, &stderr)
		got := strings.Split(strings.TrimSpace(stdout.String()), "\n")
		if code != 0 || stderr.Len() != 0 || len(got) != len(tc.lines) || saved != tc.saved {
			t.Fatalf("%v: exit %d, saved %d polls, stderr %q, stdout:\n%s", tc.args, code, saved, stderr.String(), stdout.String())
		}
		for i, want := range tc.lines {
			if !strings.Contains(got[i], want) {
				t.Errorf("%v: line %d is %q, want %q in it", tc.args, i, got[i], want)
			}
		}
	}
}

func TestExitOnEventSavesTheCursorBeforeExiting(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/activities":
			_, _ = w.Write([]byte(`{"results":[{"id":1,"object_type":"item","event_type":"completed","object_id":"t1","event_date":"` + soon + `","extra_data":{"content":"Pagar a conta de luz"}}],"next_cursor":null}`))
		case "/tasks/filter":
			_, _ = w.Write([]byte(`{"results":[],"next_cursor":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var stdout, stderr bytes.Buffer
	code := run(ctx, []string{"todoist", "--exit-on-event"}, env(map[string]string{"TODOIST_API_TOKEN": "tok", "HARK_TODOIST_API": api.URL, "HARK_NO_UPDATE": "1"}), &stdout, &stderr)

	if code != 0 || ctx.Err() != nil || stderr.Len() != 0 || !strings.Contains(stdout.String(), `todoist task.completed "Pagar a conta de luz"`) {
		t.Fatalf("exit %d, timed out %v, stderr %q, stdout:\n%s", code, ctx.Err() != nil, stderr.String(), stdout.String())
	}
	state, err := os.ReadFile(os.Getenv("XDG_STATE_HOME") + "/hark/todoist.json")
	if err != nil || !strings.Contains(string(state), soon) {
		t.Errorf("cursor not saved past the event: %v %s", err, state)
	}
}
