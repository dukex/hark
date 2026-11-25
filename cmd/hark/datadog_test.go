package main

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dukex/hark/datadog"
)

func TestDatadogNeedsKeys(t *testing.T) {
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"datadog"}, env(map[string]string{"DD_API_KEY": "k"}), nil, &stderr); code != 1 {
		t.Errorf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "set DD_APP_KEY") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestDatadogSites(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	for site, want := range map[string][2]string{
		"":                  {"https://api.datadoghq.com", "https://app.datadoghq.com"},
		"datadoghq.eu":      {"https://api.datadoghq.eu", "https://app.datadoghq.eu"},
		"us5.datadoghq.com": {"https://api.us5.datadoghq.com", "https://us5.datadoghq.com"},
	} {
		p, err := newDatadog(env(map[string]string{"DD_API_KEY": "k", "DD_APP_KEY": "a", "DD_SITE": site}), options{Filter: "service:api"})
		if err != nil {
			t.Fatal(err)
		}
		w := p.(*datadog.Watcher)
		if w.API != want[0] || w.App != want[1] || w.APIKey != "k" || w.AppKey != "a" || w.State != "/xdg/hark/datadog.json" || w.Filter != "service:api" {
			t.Errorf("%q: %+v", site, w)
		}
	}
	p, _ := newDatadog(env(map[string]string{"DD_API_KEY": "k", "DD_APP_KEY": "a", "HARK_DATADOG_API": "http://local"}), options{})
	if p.(*datadog.Watcher).API != "http://local" {
		t.Error("override ignored")
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "")
	if _, err := newDatadog(env(map[string]string{"DD_API_KEY": "k", "DD_APP_KEY": "a"}), options{}); err == nil {
		t.Error("want error without a state directory")
	}
}

func TestHarkDatadogEndToEnd(t *testing.T) {
	soon := time.Now().UTC().Add(time.Minute).Format(time.RFC3339)
	queries := make(chan string, 10)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/events" || r.Header.Get("DD-API-KEY") != "k" || r.Header.Get("DD-APPLICATION-KEY") != "a" {
			http.NotFound(w, r)
			return
		}
		queries <- r.URL.Query().Get("filter[query]")
		_, _ = w.Write([]byte(`{"data":[{"id":"e1","attributes":{"timestamp":"` + soon + `","attributes":{"title":"[Triggered] Checkout latency",
			"monitor":{"id":42,"transition":{"source_state":"OK","destination_state":"Alert"}}}}}],"meta":{"page":{}}}`))
	}))
	defer api.Close()
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout := &linesUntil{want: 1, cancel: cancel}
	var stderr bytes.Buffer
	code := run(ctx, []string{"datadog", "--filter", "service:checkout"}, env(map[string]string{"DD_API_KEY": "k", "DD_APP_KEY": "a", "HARK_DATADOG_API": api.URL}), stdout, &stderr)

	want := soon + ` datadog datadog.alert "[Triggered] Checkout latency" https://app.datadoghq.com/monitors/42` + "\n"
	if code != 0 || stderr.Len() != 0 || stdout.String() != want {
		t.Errorf("exit %d\nstderr: %s\ngot:  %swant: %s", code, stderr.String(), stdout.String(), want)
	}
	if probe := <-queries; probe != "source:alert (service:checkout)" {
		t.Errorf("probe %q", probe)
	}
}
