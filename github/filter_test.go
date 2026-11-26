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
	if q := f.terms; len(q["repo"]) != 2 || q["is"][0] != "pr" || q["reason"][1] != "review_requested" || q["org"][0] != "acme" {
		t.Errorf("%v", f)
	}
	if f, err := ParseFilter(""); err != nil || len(f.terms) != 0 {
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
	if !(Filter{}).match("any/repo", "", "issue") {
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
	w.seenEvents = nil
	if events := w.poll(context.Background()); len(events) != 0 {
		t.Errorf("reason: should only match notifications, got %v", lines(events))
	}
}

func TestFilterExclusions(t *testing.T) {
	f, err := ParseFilter("org:acme -repo:acme/legacy -reason:subscribed")
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		repo, reason string
		want         bool
	}{
		{"acme/lib", "mention", true},
		{"ACME/Legacy", "mention", false},
		{"acme/lib", "subscribed", false},
		{"dukex/foo", "mention", false},
	} {
		if got := f.match(c.repo, c.reason, "pr"); got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
	only, _ := ParseFilter("-org:acme -org:other")
	if only.match("acme/lib", "", "pr") || only.match("other/x", "", "pr") || !only.match("dukex/foo", "", "pr") {
		t.Error("repeated exclusions should each drop their matches")
	}
	for in, want := range map[string]string{
		"-label:bug":     `unknown qualifier "label:", hark knows repo, org, reason and is`,
		"-is:discussion": `is: takes issue, pr or ci, not "discussion"`,
		"-":              `"-" is not a qualifier like repo:owner/name`,
	} {
		if _, err := ParseFilter(in); err == nil || err.Error() != want {
			t.Errorf("%s: %v", in, err)
		}
	}
}

func TestExcludedThreadsStayUnread(t *testing.T) {
	routes := baseRoutes()
	routes["/notifications"] = `[` +
		thread("2", "mention", "Issue", "/repos/acme/legacy/issues/40", "", "2026-11-23T10:07:00Z") + `,` +
		thread("1", "author", "Issue", "/repos/dukex/foo/issues/3", "", "2026-11-23T10:05:00Z") + `]`
	routes["/repos/dukex/foo/issues/3"] = `{"number":3,"user":{"login":"dukex"},"html_url":"u"}`
	f := newFake(t, routes)
	w := f.watcher()
	w.Filter, _ = ParseFilter("-repo:acme/legacy")

	assertLines(t, w.poll(context.Background()), `2026-11-23T10:05:00Z github issue.updated dukex/foo#3 "author" u`)
	if strings.Join(f.marked, ",") != "1" {
		t.Errorf("marked %v, want the excluded thread left unread", f.marked)
	}
}

func TestFilterOr(t *testing.T) {
	for in, cases := range map[string]map[[2]string]bool{
		"org:acme -repo:acme/noisy OR reason:author": {
			{"acme/lib", "mention"}:   true,
			{"acme/noisy", "mention"}: false,
			{"acme/noisy", "author"}:  true,
			{"dukex/foo", "author"}:   true,
			{"dukex/foo", "mention"}:  false,
		},
		"org:acme (-repo:acme/noisy OR reason:author)": {
			{"acme/lib", "mention"}:   true,
			{"acme/noisy", "mention"}: false,
			{"acme/noisy", "author"}:  true,
			{"dukex/foo", "author"}:   false,
		},
		"(repo:dukex/foo OR (org:acme reason:mention))-reason:subscribed": {
			{"dukex/foo", "author"}:     true,
			{"dukex/foo", "subscribed"}: false,
			{"acme/lib", "mention"}:     true,
			{"acme/lib", "author"}:      false,
		},
		"repo:dukex/foo OR repo:acme/lib OR org:other": {
			{"acme/lib", ""}:  true,
			{"other/x", ""}:   true,
			{"acme/core", ""}: false,
		},
	} {
		f, err := ParseFilter(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		for c, want := range cases {
			if got := f.match(c[0], c[1], "pr"); got != want {
				t.Errorf("%s with %v: %v", in, c, got)
			}
		}
	}
	for in, want := range map[string]string{
		"OR repo:dukex/foo":                 `OR needs a qualifier on both sides`,
		"repo:dukex/foo OR":                 `OR needs a qualifier on both sides`,
		"repo:dukex/foo OR OR org:acme":     `OR needs a qualifier on both sides`,
		"(repo:dukex/foo OR) org:acme":      `OR needs a qualifier on both sides`,
		"(repo:dukex/foo OR org:acme":       `"(" has no closing ")"`,
		"repo:dukex/foo OR org:acme)":       `")" has no opening "("`,
		"org:acme ( )":                      `"()" holds no qualifier`,
		"(org:acme OR repo:dukex/foo) or:x": `unknown qualifier "or:", hark knows repo, org, reason and is`,
	} {
		if _, err := ParseFilter(in); err == nil || err.Error() != want {
			t.Errorf("%s: %v", in, err)
		}
	}
}

func TestOrKeepsMyOwnActionsThatMatchAnySide(t *testing.T) {
	routes := baseRoutes()
	routes["/users/dukex/events"] = `[` +
		activity("2", "IssuesEvent", "closed", "2026-11-23T10:05:00Z", `"issue":{"url":"{{API}}/repos/dukex/foo/issues/3"}`) + `,` +
		activity("1", "PullRequestEvent", "opened", "2026-11-23T10:02:00Z", `"pull_request":{"url":"{{API}}/repos/dukex/foo/pulls/12"}`) + `]`
	routes["/repos/dukex/foo/pulls/12"] = `{"number":12,"title":"feat","html_url":"p12"}`
	f := newFake(t, routes)
	w := f.watcher()
	w.Filter, _ = ParseFilter("reason:mention OR is:pr")

	assertLines(t, w.poll(context.Background()), `2026-11-23T10:02:00Z github pr.opened dukex/foo#12 @dukex "feat" p12`)
}
