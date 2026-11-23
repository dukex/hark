package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type linesUntil struct {
	bytes.Buffer
	want   int
	cancel context.CancelFunc
}

func (l *linesUntil) Write(p []byte) (int, error) {
	n, err := l.Buffer.Write(p)
	if strings.Count(l.String(), "\n") >= l.want {
		l.cancel()
	}
	return n, err
}

func TestHarkGitHubEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	routes := map[string]string{
		"/user": `{"login":"dukex"}`,
		"/notifications": `[
			{"id":"2","reason":"subscribed","updated_at":"SOON","repository":{"full_name":"acme/lib"},
			 "subject":{"type":"PullRequest","url":"API/repos/acme/lib/pulls/40","latest_comment_url":"API/repos/acme/lib/issues/comments/70"}},
			{"id":"1","reason":"author","updated_at":"SOON","repository":{"full_name":"dukex/foo"},
			 "subject":{"type":"Issue","url":"API/repos/dukex/foo/issues/3","latest_comment_url":null}}]`,
		"/repos/dukex/foo/issues/3":          `{"number":3,"user":{"login":"dukex"},"closed_at":"SOON","closed_by":{"login":"carol"},"html_url":"https://github.com/dukex/foo/issues/3"}`,
		"/repos/acme/lib/pulls/40":           `{"number":40,"user":{"login":"erin"},"html_url":"https://github.com/acme/lib/pull/40"}`,
		"/repos/acme/lib/issues/comments/70": `{"user":{"login":"frank"},"body":"rebase?","created_at":"SOON","html_url":"https://github.com/acme/lib/pull/40#issuecomment-70"}`,
		"/users/dukex/events": `[{"id":"9","type":"PullRequestEvent","created_at":"SOON","repo":{"name":"dukex/foo"},
			"payload":{"action":"opened","pull_request":{"number":12,"title":"feat: x","html_url":"https://github.com/dukex/foo/pull/12"}}}]`,
	}
	var api *httptest.Server
	api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, ok := routes[r.URL.Path]
		if !ok || r.Header.Get("Authorization") != "Bearer secret" {
			http.NotFound(w, r)
			return
		}
		w.Write([]byte(strings.NewReplacer("SOON", soon, "API", api.URL).Replace(body)))
	}))
	defer api.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 3, cancel: cancel}
	var stderr bytes.Buffer
	code := run(ctx, []string{"github"}, env(map[string]string{"GITHUB_TOKEN": "secret", "HARK_GITHUB_API": api.URL}), stdout, &stderr)

	want := strings.ReplaceAll(`SOON github issue.closed dukex/foo#3 @carol https://github.com/dukex/foo/issues/3
SOON github pr.commented acme/lib#40 @frank watching "rebase?" https://github.com/acme/lib/pull/40#issuecomment-70
SOON github pr.opened dukex/foo#12 @dukex "feat: x" https://github.com/dukex/foo/pull/12
`, "SOON", soon)
	if code != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Errorf("exit %d\nstderr: %s\ngot:\n%s\nwant:\n%s", code, stderr.String(), stdout.String(), want)
	}
}
