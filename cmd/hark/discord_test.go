package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark/discord"
)

func TestDiscordNeedsTokenAndUser(t *testing.T) {
	for _, vars := range []map[string]string{nil, {"DISCORD_TOKEN": "tok"}, {"DISCORD_USER_ID": "U1"}} {
		var stderr bytes.Buffer
		if code := run(context.Background(), []string{"discord"}, env(vars), nil, &stderr); code != 1 {
			t.Errorf("%v: exit %d", vars, code)
		}
		if !strings.Contains(stderr.String(), "set DISCORD_TOKEN and DISCORD_USER_ID") {
			t.Errorf("%v: stderr %q", vars, stderr.String())
		}
	}
}

func TestDiscordConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	vars := map[string]string{"DISCORD_TOKEN": "tok", "DISCORD_USER_ID": "U1"}
	p, err := newDiscord(env(vars), options{Filter: "author_type=-bot"})
	if err != nil {
		t.Fatal(err)
	}
	if w := p.(*discord.Watcher); w.API != "https://discord.com/api/v10" || w.Token != "tok" || w.UserID != "U1" || w.State != "/xdg/hark/discord.json" || w.Filter != "author_type=-bot" {
		t.Errorf("%+v", w)
	}
	vars["HARK_DISCORD_API"] = "http://local"
	p, _ = newDiscord(env(vars), options{})
	if p.(*discord.Watcher).API != "http://local" {
		t.Error("override ignored")
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newDiscord(env(vars), options{}); err == nil {
		t.Error("want error without a state directory")
	}
}

func TestHarkDiscordEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	searches := make(chan string, 10)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bot tok" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/users/@me":
			_, _ = w.Write([]byte(`{"id":"B1","username":"harkbot"}`))
		case "/users/U1":
			_, _ = w.Write([]byte(`{"id":"U1","username":"duke"}`))
		case "/users/@me/guilds":
			_, _ = w.Write([]byte(`[{"id":"G1","name":"Acme Dev"}]`))
		case "/guilds/G1/messages/search":
			searches <- r.URL.RawQuery
			if r.URL.Query().Has("replied_to_user_id") {
				_, _ = w.Write([]byte(`{"total_results":0,"messages":[]}`))
				return
			}
			_, _ = w.Write([]byte(`{"total_results":1,"messages":[[{"id":"M1","channel_id":"C1","author":{"id":"U2","username":"ana"},
				"content":"<@U1> can you look?","timestamp":"` + soon.Format(time.RFC3339) + `"}]]}`))
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
	code := run(ctx, []string{"discord", "--filter", "author_type=-bot"}, env(map[string]string{"DISCORD_TOKEN": "tok", "DISCORD_USER_ID": "U1", "HARK_DISCORD_API": api.URL}), stdout, &stderr)

	want := soon.Format(time.RFC3339) + ` discord discord.mentioned @ana "Acme Dev: <@U1> can you look?" https://discord.com/channels/G1/C1/M1` + "\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Errorf("exit %d\nstderr: %s\ngot:  %swant: %s", code, stderr.String(), stdout.String(), want)
	}
	if probe, search := <-searches, <-searches; !strings.Contains(probe, "limit=1") || !strings.Contains(probe, "author_type=-bot") || !strings.Contains(search, "min_id=") || !strings.Contains(search, "author_type=-bot") {
		t.Errorf("probe %s, search %s", probe, search)
	}
}
