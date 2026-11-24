package github

import (
	"fmt"
	"slices"
	"strings"
)

// Filter keeps only the threads that match every qualifier, and any of the values given for it.
// Qualifiers written with a leading "-", like -repo:, drop the threads that match any of their values.
type Filter map[string][]string

var reasons = []string{
	"approval_requested", "assign", "author", "ci_activity", "comment", "invitation", "manual", "member_feature_requested",
	"mention", "review_requested", "security_advisory_credit", "security_alert", "state_change", "subscribed", "team_mention",
}

// ParseFilter reads qualifiers from github.com notifications search: repo:, org:, reason: and is:issue|pr|ci,
// each of them negated with a leading "-".
func ParseFilter(s string) (Filter, error) {
	f := Filter{}
	for _, term := range strings.Fields(s) {
		key, value, ok := strings.Cut(term, ":")
		key, exclude := strings.CutPrefix(key, "-")
		switch {
		case !ok || value == "":
			return nil, fmt.Errorf("%q is not a qualifier like repo:owner/name", term)
		case !slices.Contains([]string{"repo", "org", "reason", "is"}, key):
			return nil, fmt.Errorf("unknown qualifier %q, hark knows repo, org, reason and is", key+":")
		case key == "is" && !slices.Contains([]string{"issue", "pr", "ci"}, value):
			return nil, fmt.Errorf("is: takes issue, pr or ci, not %q", value)
		case key == "reason":
			value = strings.ReplaceAll(value, "-", "_")
			if !slices.Contains(reasons, value) {
				return nil, fmt.Errorf("unknown reason %q, see docs.github.com/rest/activity/notifications#about-notification-reasons", value)
			}
		}
		if exclude {
			key = "-" + key
		}
		f[key] = append(f[key], value)
	}
	return f, nil
}

func (f Filter) match(repo, reason, is string) bool {
	org, _, _ := strings.Cut(repo, "/")
	got := map[string]string{"repo": repo, "org": org, "reason": reason, "is": is}
	for key, values := range f {
		name, exclude := strings.CutPrefix(key, "-")
		if slices.ContainsFunc(values, func(v string) bool { return strings.EqualFold(v, got[name]) }) == exclude {
			return false
		}
	}
	return true
}
