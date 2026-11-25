package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark/linear"
)

func TestLinearNeedsKey(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"linear"}, env(nil), nil, &stderr); code != 1 {
		t.Errorf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "set LINEAR_API_KEY") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestLinearConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	p, err := newLinear(env(map[string]string{"LINEAR_API_KEY": "lin_api_1"}), options{Filter: `{"type":{"eq":"issueMention"}}`})
	if err != nil {
		t.Fatal(err)
	}
	if w := p.(*linear.Watcher); w.API != "https://api.linear.app/graphql" || w.Key != "lin_api_1" || w.State != "/xdg/hark/linear.json" || w.Filter != `{"type":{"eq":"issueMention"}}` {
		t.Errorf("%+v", w)
	}
	p, _ = newLinear(env(map[string]string{"LINEAR_API_KEY": "lin_api_1", "HARK_LINEAR_API": "http://local"}), options{})
	if p.(*linear.Watcher).API != "http://local" {
		t.Error("override ignored")
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newLinear(env(map[string]string{"LINEAR_API_KEY": "lin_api_1"}), options{}); err == nil {
		t.Error("want error without a state directory")
	}
}

func TestHarkLinearEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	filters := make(chan string, 10)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "lin_api_1" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"errors":[{"message":"Authentication required"}]}`))
			return
		}
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if strings.Contains(req.Query, "viewer") {
			_, _ = w.Write([]byte(`{"data":{"viewer":{"name":"Duke"}}}`))
			return
		}
		filter, _ := json.Marshal(req.Variables["filter"])
		filters <- string(filter)
		_, _ = w.Write([]byte(`{"data":{"notifications":{"nodes":[{"id":"n1","type":"issueCommentMention","createdAt":"` + soon.Format(time.RFC3339) + `",
			"actor":{"displayName":"ana"},"issue":{"identifier":"ENG-7","title":"Fix login","url":"https://linear.app/acme/issue/ENG-7/fix-login","state":{"name":"Todo"}},
			"comment":{"body":"@duke can you check?","url":"https://linear.app/acme/issue/ENG-7/fix-login#comment-1"}}],"pageInfo":{"hasNextPage":false,"endCursor":""}}}}`))
	}))
	defer api.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 1, cancel: cancel}
	var stderr bytes.Buffer
	filter := `{"type":{"eq":"issueCommentMention"}}`
	code := run(ctx, []string{"linear", "--filter", filter}, env(map[string]string{"LINEAR_API_KEY": "lin_api_1", "HARK_LINEAR_API": api.URL}), stdout, &stderr)

	want := soon.Format(time.RFC3339) + ` linear linear.mentioned ENG-7 @ana "@duke can you check?" https://linear.app/acme/issue/ENG-7/fix-login#comment-1` + "\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Errorf("exit %d\nstderr: %s\ngot:  %swant: %s", code, stderr.String(), stdout.String(), want)
	}
	if probe, poll := <-filters, <-filters; probe != filter || !strings.HasPrefix(poll, `{"and":[{"createdAt":{"gte":"`) || !strings.HasSuffix(poll, `"}},`+filter+`]}`) {
		t.Errorf("probe %s, poll %s", probe, poll)
	}
}
