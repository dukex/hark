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
			"payload":{"action":"opened","number":12,"pull_request":{"number":12,"url":"API/repos/dukex/foo/pulls/12"}}}]`,
		"/repos/dukex/foo/pulls/12": `{"number":12,"title":"feat: x","html_url":"https://github.com/dukex/foo/pull/12"}`,
	}
	var marked []string
	var api *httptest.Server
	api = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if id, ok := strings.CutPrefix(r.URL.Path, "/notifications/threads/"); ok && r.Method == http.MethodPatch {
			marked = append(marked, id)
			w.WriteHeader(http.StatusResetContent)
			return
		}
		body, ok := routes[r.URL.Path]
		if !ok || r.Header.Get("Authorization") != "Bearer secret" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(strings.NewReplacer("SOON", soon, "API", api.URL).Replace(body)))
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
	if strings.Join(marked, ",") != "1,2" {
		t.Errorf("marked %v, want both threads read", marked)
	}
}

func TestHarkTodoistEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/activities":
			_, _ = w.Write([]byte(`{"results":[{"id":99999999999999999999999999999999999999,"object_type":"item","event_type":"completed","object_id":"t1","event_date":"` + soon + `","extra_data":{"content":"Pagar a conta de luz"}}],"next_cursor":null}`))
		case "/tasks/filter":
			_, _ = w.Write([]byte(`{"results":[{"id":"t2","content":"Pagar boleto"}],"next_cursor":null}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer api.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 2, cancel: cancel}
	var stderr bytes.Buffer
	code := run(ctx, []string{"todoist"}, env(map[string]string{"TODOIST_API_TOKEN": "tok", "HARK_TODOIST_API": api.URL}), stdout, &stderr)

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if code != 0 || stderr.Len() != 0 || len(lines) != 2 ||
		lines[0] != soon+` todoist task.completed "Pagar a conta de luz" https://app.todoist.com/app/task/t1` ||
		!strings.HasSuffix(lines[1], ` todoist task.due "Pagar boleto" https://app.todoist.com/app/task/t2`) {
		t.Errorf("exit %d\nstderr: %s\nstdout:\n%s", code, stderr.String(), stdout.String())
	}
	if _, err := os.Stat(os.Getenv("XDG_STATE_HOME") + "/hark/todoist.json"); err != nil {
		t.Errorf("state not saved: %v", err)
	}
}
