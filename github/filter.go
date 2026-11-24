package github

import (
	"fmt"
	"slices"
	"strings"
)

// Filter keeps only the threads that match every qualifier, and any of the values given for it.
type Filter map[string][]string

// ParseFilter reads qualifiers from github.com notifications search: repo:, org:, reason: and is:issue|pr|ci.
func ParseFilter(s string) (Filter, error) {
	f := Filter{}
	for _, term := range strings.Fields(s) {
		key, value, ok := strings.Cut(term, ":")
		switch {
		case !ok || value == "":
			return nil, fmt.Errorf("%q is not a qualifier like repo:owner/name", term)
		case !slices.Contains([]string{"repo", "org", "reason", "is"}, key):
			return nil, fmt.Errorf("unknown qualifier %q, hark knows repo, org, reason and is", key+":")
		case key == "is" && !slices.Contains([]string{"issue", "pr", "ci"}, value):
			return nil, fmt.Errorf("is: takes issue, pr or ci, not %q", value)
		}
		f[key] = append(f[key], value)
	}
	return f, nil
}

func (f Filter) match(repo, reason, is string) bool {
	org, _, _ := strings.Cut(repo, "/")
	got := map[string]string{"repo": repo, "org": org, "reason": reason, "is": is}
	for key, values := range f {
		if !slices.ContainsFunc(values, func(v string) bool { return strings.EqualFold(v, got[key]) }) {
			return false
		}
	}
	return true
}
