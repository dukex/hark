package hark

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Provider watches an external service and calls emit once per event until ctx is done.
type Provider interface {
	Watch(ctx context.Context, emit func(Event)) error
}

// Event is one thing that happened on a provider, printed as a single line.
type Event struct {
	Time     time.Time
	Provider string
	Kind     string
	Repo     string
	Number   int
	Actor    string
	Watching bool
	Text     string
	URL      string
}

const maxText = 140

func (e Event) String() string {
	parts := []string{e.Time.UTC().Format(time.RFC3339), e.Provider, e.Kind}
	if e.Number != 0 {
		parts = append(parts, fmt.Sprintf("%s#%d", e.Repo, e.Number))
	} else if e.Repo != "" {
		parts = append(parts, e.Repo)
	}
	if e.Actor != "" {
		parts = append(parts, "@"+e.Actor)
	}
	if e.Watching {
		parts = append(parts, "watching")
	}
	if e.Text != "" {
		parts = append(parts, fmt.Sprintf("%q", oneLine(e.Text)))
	}
	if e.URL != "" {
		parts = append(parts, e.URL)
	}
	return strings.Join(parts, " ")
}

func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > maxText {
		return string(r[:maxText]) + "…"
	}
	return s
}
