package github

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Filter keeps only the threads that match every qualifier, and any of the values given for it.
// Qualifiers written with a leading "-", like -repo:, drop the threads that match any of their values.
// OR keeps the threads that match either side, and parentheses group qualifiers.
type Filter struct {
	or    []Filter
	terms map[string][]string
	and   []Filter
}

var reasons = []string{
	"approval_requested", "assign", "author", "ci_activity", "comment", "invitation", "manual", "member_feature_requested",
	"mention", "review_requested", "security_advisory_credit", "security_alert", "state_change", "subscribed", "team_mention",
}

var errOr = errors.New("OR needs a qualifier on both sides")

// ParseFilter reads qualifiers from github.com notifications search: repo:, org:, reason: and is:issue|pr|ci,
// each of them negated with a leading "-", joined by OR and grouped with parentheses.
func ParseFilter(s string) (Filter, error) {
	tokens := strings.Fields(strings.NewReplacer("(", " ( ", ")", " ) ").Replace(s))
	f, rest, err := parseOr(tokens)
	if err == nil && len(rest) > 0 {
		err = errors.New(`")" has no opening "("`)
	}
	return f, err
}

func parseOr(tokens []string) (Filter, []string, error) {
	var sides []Filter
	for {
		f, rest, err := parseAnd(tokens)
		switch {
		case err != nil:
			return Filter{}, nil, err
		case len(rest) == len(tokens) && (len(sides) > 0 || len(rest) > 0 && rest[0] == "OR"):
			return Filter{}, nil, errOr
		}
		sides = append(sides, f)
		if len(rest) == 0 || rest[0] != "OR" {
			if len(sides) == 1 {
				return f, rest, nil
			}
			return Filter{or: sides}, rest, nil
		}
		tokens = rest[1:]
	}
}

func parseAnd(tokens []string) (Filter, []string, error) {
	f := Filter{terms: map[string][]string{}}
	for len(tokens) > 0 && tokens[0] != "OR" && tokens[0] != ")" {
		term := tokens[0]
		tokens = tokens[1:]
		if term == "(" {
			group, rest, err := parseOr(tokens)
			switch {
			case err != nil:
				return Filter{}, nil, err
			case len(rest) == 0:
				return Filter{}, nil, errors.New(`"(" has no closing ")"`)
			case len(rest) == len(tokens):
				return Filter{}, nil, errors.New(`"()" holds no qualifier`)
			}
			f.and = append(f.and, group)
			tokens = rest[1:]
			continue
		}
		key, value, err := parseTerm(term)
		if err != nil {
			return Filter{}, nil, err
		}
		f.terms[key] = append(f.terms[key], value)
	}
	return f, tokens, nil
}

func parseTerm(term string) (string, string, error) {
	key, value, ok := strings.Cut(term, ":")
	key, exclude := strings.CutPrefix(key, "-")
	switch {
	case !ok || value == "":
		return "", "", fmt.Errorf("%q is not a qualifier like repo:owner/name", term)
	case !slices.Contains([]string{"repo", "org", "reason", "is"}, key):
		return "", "", fmt.Errorf("unknown qualifier %q, hark knows repo, org, reason and is", key+":")
	case key == "is" && !slices.Contains([]string{"issue", "pr", "ci"}, value):
		return "", "", fmt.Errorf("is: takes issue, pr or ci, not %q", value)
	case key == "reason":
		value = strings.ReplaceAll(value, "-", "_")
		if !slices.Contains(reasons, value) {
			return "", "", fmt.Errorf("unknown reason %q, see docs.github.com/rest/activity/notifications#about-notification-reasons", value)
		}
	}
	if exclude {
		key = "-" + key
	}
	return key, value, nil
}

func (f Filter) match(repo, reason, is string) bool {
	if f.or != nil {
		return slices.ContainsFunc(f.or, func(side Filter) bool { return side.match(repo, reason, is) })
	}
	org, _, _ := strings.Cut(repo, "/")
	got := map[string]string{"repo": repo, "org": org, "reason": reason, "is": is}
	for key, values := range f.terms {
		name, exclude := strings.CutPrefix(key, "-")
		if slices.ContainsFunc(values, func(v string) bool { return strings.EqualFold(v, got[name]) }) == exclude {
			return false
		}
	}
	return !slices.ContainsFunc(f.and, func(group Filter) bool { return !group.match(repo, reason, is) })
}
