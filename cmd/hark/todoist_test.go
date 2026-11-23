package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/dukex/hark/todoist"
)

func TestTodoistNeedsToken(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"todoist"}, env(nil), nil, &stderr); code != 1 {
		t.Errorf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "set TODOIST_API_TOKEN") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestTodoistDefaults(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	p, err := newTodoist(env(map[string]string{"TODOIST_API_TOKEN": "tok"}))
	if err != nil {
		t.Fatal(err)
	}
	w := p.(*todoist.Watcher)
	if w.API != "https://api.todoist.com/api/v1" || w.Token != "tok" || w.State != "/xdg/hark/todoist.json" {
		t.Errorf("%+v", w)
	}
	p, _ = newTodoist(env(map[string]string{"TODOIST_API_TOKEN": "tok", "HARK_TODOIST_API": "http://local"}))
	if p.(*todoist.Watcher).API != "http://local" {
		t.Errorf("override ignored")
	}
}

func TestTodoistWithoutStateDir(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newTodoist(env(map[string]string{"TODOIST_API_TOKEN": "tok"})); err == nil {
		t.Error("want error without a state directory")
	}
}
