package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark/jira"
)

func TestJiraNeedsCredentials(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"jira"}, env(map[string]string{"JIRA_EMAIL": "e"}), nil, &stderr); code != 1 {
		t.Errorf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "set JIRA_URL, JIRA_API_TOKEN") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestJiraConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	p, err := newJira(env(map[string]string{"JIRA_URL": "https://example.atlassian.net", "JIRA_EMAIL": "e", "JIRA_API_TOKEN": "t"}), options{})
	if err != nil {
		t.Fatal(err)
	}
	if w := p.(*jira.Watcher); w.Site != "https://example.atlassian.net" || w.Email != "e" || w.Token != "t" || w.State != "/xdg/hark/jira.json" {
		t.Errorf("%+v", w)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newJira(env(map[string]string{"JIRA_URL": "u", "JIRA_EMAIL": "e", "JIRA_API_TOKEN": "t"}), options{}); err == nil {
		t.Error("want error without a state directory")
	}
}

func TestHarkJiraEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if user, pass, ok := r.BasicAuth(); !ok || user != "e" || pass != "t" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/rest/api/3/myself":
			_, _ = w.Write([]byte(`{"accountId":"me"}`))
		case "/rest/api/3/issue/PROJ-7/comment":
			_, _ = w.Write([]byte(`{"comments":[{"id":"9","author":{"accountId":"bob","displayName":"Bob"},"created":"2020-01-01T00:00:00.000+0000","body":{"type":"doc","content":[]}}]}`))
		case "/rest/api/3/search/jql":
			_, _ = w.Write([]byte(`{"issues":[{"key":"PROJ-7","fields":{"summary":"Checkout","comment":{"comments":[]}},
				"changelog":{"histories":[{"id":"1","author":{"accountId":"ana","displayName":"Ana"},"created":"` +
				soon.Format("2006-01-02T15:04:05.000-0700") + `","items":[{"field":"status","fromString":"To Do","toString":"Done"}]}]}}],"isLast":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer site.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 1, cancel: cancel}
	var stderr bytes.Buffer
	code := run(ctx, []string{"jira"}, env(map[string]string{"JIRA_URL": site.URL, "JIRA_EMAIL": "e", "JIRA_API_TOKEN": "t"}), stdout, &stderr)

	want := soon.Format(time.RFC3339) + ` jira jira.transitioned PROJ-7 "Checkout: To Do to Done (Ana)" ` + site.URL + "/browse/PROJ-7\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Errorf("exit %d\nstderr: %s\ngot:  %swant: %s", code, stderr.String(), stdout.String(), want)
	}
}
