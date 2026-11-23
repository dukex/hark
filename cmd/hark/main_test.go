package main

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/dukex/hark/github"
)

func env(vars map[string]string) func(string) string {
	return func(k string) string { return vars[k] }
}

func TestUsage(t *testing.T) {
	for _, args := range [][]string{nil, {"gitlab"}, {"github", "extra"}} {
		var stderr bytes.Buffer
		if code := run(context.Background(), args, env(nil), nil, &stderr); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
		if stderr.String() != "usage: hark <github|jira|todoist>\n" {
			t.Errorf("%v: stderr %q", args, stderr.String())
		}
	}
}

func TestMainExitsWithRunCode(t *testing.T) {
	defer func(args []string) { os.Args, exit = args, os.Exit }(os.Args)
	os.Args = []string{"hark"}
	var code int
	exit = func(c int) { code = c }
	main()
	if code != 2 {
		t.Errorf("exit %d, want 2", code)
	}
}

func TestTokenFallsBackToGh(t *testing.T) {
	defer func(f func() (string, error)) { ghToken = f }(ghToken)
	var auth string
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		http.Error(w, "stop", http.StatusUnauthorized)
	}))
	defer api.Close()
	ghToken = func() (string, error) { return "from-gh", nil }

	var stderr bytes.Buffer
	code := run(context.Background(), []string{"github"}, env(map[string]string{"HARK_GITHUB_API": api.URL}), nil, &stderr)
	if code != 1 || auth != "Bearer from-gh" || !strings.Contains(stderr.String(), "401") {
		t.Errorf("code %d, auth %q, stderr %q", code, auth, stderr.String())
	}
}

func TestMissingTokenIsReported(t *testing.T) {
	defer func(f func() (string, error)) { ghToken = f }(ghToken)
	ghToken = func() (string, error) { return "", errors.New("not logged in") }

	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"github"}, env(nil), nil, &stderr); code != 1 {
		t.Errorf("exit %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "set GITHUB_TOKEN or log in with gh: not logged in") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestDefaultAPIIsGitHubCom(t *testing.T) {
	p, err := newGitHub(env(map[string]string{"GITHUB_TOKEN": "t"}))
	if err != nil {
		t.Fatal(err)
	}
	if w := p.(*github.Watcher); w.API != "https://api.github.com" || w.Token != "t" {
		t.Errorf("api %q token %q", w.API, w.Token)
	}
}

func TestGhTokenRunsGhCLI(t *testing.T) {
	token, err := ghToken()
	if err == nil && (token == "" || strings.ContainsAny(token, " \n")) {
		t.Errorf("gh succeeded but returned %q", token)
	}
}
