package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/dukex/hark"
)

type scripted []hark.Event

func (s scripted) Watch(_ context.Context, emit func(hark.Event)) error {
	for _, e := range s {
		emit(e)
	}
	return nil
}

func (scripted) Doctor(context.Context) []hark.Check { return nil }

func TestRepeatedErrorsArePrintedOnce(t *testing.T) {
	failure := func(text string) hark.Event { return hark.Event{Provider: "todoist", Kind: "error", Text: text} }
	activityDown, filterDown := failure("GET /api/v1/activities: 500"), failure("GET /api/v1/tasks/filter: 500")
	done := hark.Event{Provider: "todoist", Kind: "task.completed", Text: "done"}
	old, oldVersion := providers, version
	t.Cleanup(func() { providers, version = old, oldVersion })
	version = "dev"
	providers = map[string]func(func(string) string) (hark.Provider, error){
		"todoist": func(func(string) string) (hark.Provider, error) {
			return scripted{activityDown, filterDown, activityDown, filterDown, done, activityDown, activityDown}, nil
		},
	}
	var stdout bytes.Buffer
	if code := run(context.Background(), []string{"todoist"}, env(nil), &stdout, &bytes.Buffer{}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	want := `0001-01-01T00:00:00Z todoist error "GET /api/v1/activities: 500"
0001-01-01T00:00:00Z todoist error "GET /api/v1/tasks/filter: 500"
0001-01-01T00:00:00Z todoist task.completed "done"
0001-01-01T00:00:00Z todoist error "GET /api/v1/activities: 500"
`
	if stdout.String() != want {
		t.Errorf("got:\n%s\nwant:\n%s", stdout.String(), want)
	}
}
