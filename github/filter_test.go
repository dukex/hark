package github

import (
	"context"
	"strings"
	"testing"
)

func TestParseFilter(t *testing.T) {
	f, err := ParseFilter("repo:dukex/foo  repo:acme/lib is:pr reason:mention reason:review-requested org:acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(f["repo"]) != 2 || f["is"][0] != "pr" || f["reason"][1] != "review_requested" || f["org"][0] != "acme" {
		t.Errorf("%v", f)
	}
	if f, err := ParseFilter(""); err != nil || len(f) != 0 {
		t.Errorf("empty: %v, %v", f, err)
	}
	for in, want := range map[string]string{
		"dukex/foo":      `"dukex/foo" is not a qualifier like repo:owner/name`,
		"repo:":          `"repo:" is not a qualifier like repo:owner/name`,
		"label:bug":      `unknown qualifier "label:", hark knows repo, org, reason and is`,
		"is:discussion":  `is: takes issue, pr or ci, not "discussion"`,
		"reason:mentoin": `unknown reason "mentoin", see docs.github.com/rest/activity/notifications#about-notification-reasons`,
	} {
		if _, err := ParseFilter(in); err == nil || err.Error() != want {
			t.Errorf("%s: %v", in, err)
		}
	}
}

func TestFilterMatches(t *testing.T) {
	f, _ := ParseFilter("org:ACME is:pr is:ci")
	for _, c := range []struct {
		repo, reason, is string
		want             bool
	}{
		{"acme/lib", "author", "pr", true},
		{"acme/lib", "", "ci", true},
		{"acme/lib", "author", "issue", false},
		{"dukex/foo", "author", "pr", false},
	} {
		if got := f.match(c.repo, c.reason, c.is); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
	if !Filter(nil).match("any/repo", "", "issue") {
		t.Error("an empty filter keeps everything")
	}
}

func TestFilteredThreadsStayUnread(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		thread("3", "mention", "Issue", "/repos/acme/lib/issues/40", "", "2026-11-23T10:07:00Z") + `,` +
		ciThread("2", "ci_activity", "CheckSuite", "CI workflow run failed for main branch", "2026-11-23T10:06:00Z") + `,` +
		thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	f := newFake(t, routes)
	w := f.watcher()
	w.Filter, _ = ParseFilter("repo:dukex/foo is:issue")

	assertLines(t, w.poll(context.Background()), `2026-11-23T10:05:00Z github issue.updated dukex/foo#3 "author" u`)
	if strings.Join(f.marked, ",") != "1" {
		t.Errorf("marked %v, want only the matching thread", f.marked)
	}
}

func TestFilterAppliesToMyOwnActions(t *testing.T) {
	routes := baseRoutes()
	routes["/users/dukex/events"] = `[` +
		activity("2", "IssuesEvent", "closed", "2026-11-23T10:05:00Z", `"issue":{"url":"{{API}}/repos/dukex/foo/issues/3"}`) + `,` +
		activity("1", "PullRequestEvent", "opened", "2026-11-23T10:02:00Z", `"pull_request":{"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"title":"feat","html_url":"p12"}`
	f := newFake(t, routes)
	w := f.watcher()
	w.Filter, _ = ParseFilter("is:pr")

	assertLines(t, w.poll(context.Background()), `2026-11-23T10:02:00Z github pr.opened dukex/foo#12 @dukex "feat" p12`)
	if f.hits["/repos/dukex/foo/issues/3"] != 0 {
		t.Error("fetched an issue the filter drops")
	}
	w.Filter, _ = ParseFilter("reason:mention")
	w.lastEvent = 0
	if events := w.poll(context.Background()); len(events) != 0 {
		t.Errorf("reason: should only match notifications, got %v", lines(events))
	}
}
