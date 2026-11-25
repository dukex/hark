package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark/slack"
)

func TestSlackNeedsToken(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"slack"}, env(nil), nil, &stderr); code != 1 {
		t.Errorf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "set SLACK_TOKEN") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestSlackConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	p, err := newSlack(env(map[string]string{"SLACK_TOKEN": "xoxp-1"}), options{Filter: "-in:#random"})
	if err != nil {
		t.Fatal(err)
	}
	if w := p.(*slack.Watcher); w.API != "https://slack.com/api" || w.Token != "xoxp-1" || w.State != "/xdg/hark/slack.json" || w.Filter != "-in:#random" {
		t.Errorf("%+v", w)
	}
	p, _ = newSlack(env(map[string]string{"SLACK_TOKEN": "xoxp-1", "HARK_SLACK_API": "http://local"}), options{})
	if p.(*slack.Watcher).API != "http://local" {
		t.Error("override ignored")
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newSlack(env(map[string]string{"SLACK_TOKEN": "xoxp-1"}), options{}); err == nil {
		t.Error("want error without a state directory")
	}
}

func TestHarkSlackEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	ts := strconv.FormatInt(soon.Unix(), 10) + ".000200"
	queries := make(chan string, 10)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer xoxp-1" {
			_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
			return
		}
		switch r.URL.Path {
		case "/auth.test":
			_, _ = w.Write([]byte(`{"ok":true,"user_id":"U1","user":"duke","team":"Acme"}`))
		case "/search.messages":
			queries <- r.URL.Query().Get("query")
			if strings.HasPrefix(r.URL.Query().Get("query"), "to:me") {
				_, _ = w.Write([]byte(`{"ok":true,"messages":{"matches":[],"paging":{"pages":1}}}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true,"messages":{"matches":[{"type":"message","channel":{"id":"C1","name":"deploys"},"user":"U2","username":"ana",
				"ts":"` + ts + `","text":"<@U1> can you look?","permalink":"https://example.slack.com/archives/C1/p1"}],"paging":{"pages":1}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 1, cancel: cancel}
	var stderr bytes.Buffer
	code := run(ctx, []string{"slack", "--filter", "-in:#random"}, env(map[string]string{"SLACK_TOKEN": "xoxp-1", "HARK_SLACK_API": api.URL}), stdout, &stderr)

	want := soon.Format(time.RFC3339) + ` slack slack.mentioned #deploys @ana "<@U1> can you look?" https://example.slack.com/archives/C1/p1` + "\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Errorf("exit %d\nstderr: %s\ngot:  %swant: %s", code, stderr.String(), stdout.String(), want)
	}
	if probe, search := <-queries, <-queries; probe != "<@U1> -in:#random" || !strings.HasPrefix(search, "<@U1> after:") || !strings.HasSuffix(search, " -in:#random") {
		t.Errorf("probe %q, search %q", probe, search)
	}
}
