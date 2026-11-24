package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark"
	"github.com/dukex/hark/github"
	"github.com/dukex/hark/jira"
	"github.com/dukex/hark/todoist"
)

func TestSinceIsPassedToTheProvider(t *testing.T) {
	var got options
	withProviders(t, map[string]constructor{
		"github": func(_ func(string) string, o options) (hark.Provider, error) {
			got = o
			return fakeProvider{}, nil
		},
	})
	for arg, want := range map[string]time.Time{
		"2026-11-23T07:00:00Z":      time.Date(2026, 11, 23, 7, 0, 0, 0, time.UTC),
		"2026-11-23T07:00:00-03:00": time.Date(2026, 11, 23, 10, 0, 0, 0, time.UTC),
		"2026-11-23":                time.Date(2026, 11, 23, 0, 0, 0, 0, time.Local),
	} {
		got = options{}
		var stderr bytes.Buffer
		if code := run(context.Background(), []string{"github", "--since", arg}, env(map[string]string{"HARK_NO_UPDATE": "1"}), nil, &stderr); code != 0 || !got.Since.Equal(want) {
			t.Errorf("%s: exit %d, since %v, stderr %q", arg, code, got.Since, stderr.String())
		}
	}
}

func TestBadSinceIsAUsageError(t *testing.T) {
	for _, args := range [][]string{{"github", "--since", "yesterday"}, {"jira", "--since"}, {"todoist", "--until", "x"}} {
		var stderr bytes.Buffer
		if code := run(context.Background(), args, env(nil), nil, &stderr); code != 2 {
			t.Errorf("%v: exit %d", args, code)
		}
		if !strings.HasSuffix(stderr.String(), "usage: hark <github|jira|todoist> [--since TIME]\n") {
			t.Errorf("%v: stderr %q", args, stderr.String())
		}
	}
	var stderr bytes.Buffer
	run(context.Background(), []string{"github", "--since", "yesterday"}, env(nil), nil, &stderr)
	if !strings.Contains(stderr.String(), "want a time like 2026-11-23T10:05:00Z or a date like 2026-11-23") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestConstructorsSetSince(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	o := options{Since: time.Date(2026, 11, 23, 7, 0, 0, 0, time.UTC)}
	gh, _ := newGitHub(env(map[string]string{"GITHUB_TOKEN": "t"}), o)
	jr, _ := newJira(env(map[string]string{"JIRA_URL": "u", "JIRA_EMAIL": "e", "JIRA_API_TOKEN": "t"}), o)
	td, _ := newTodoist(env(map[string]string{"TODOIST_API_TOKEN": "t"}), o)
	for name, since := range map[string]time.Time{
		"github":  gh.(*github.Watcher).Since,
		"jira":    jr.(*jira.Watcher).Since,
		"todoist": td.(*todoist.Watcher).Since,
	} {
		if !since.Equal(o.Since) {
			t.Errorf("%s: since %v", name, since)
		}
	}
}
