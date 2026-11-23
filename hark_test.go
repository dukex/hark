package hark

import (
	"strings"
	"testing"
	"time"
)

func TestEventString(t *testing.T) {
	at := time.Date(2026, 11, 23, 10, 2, 11, 0, time.FixedZone("BRT", -3*3600))
	cases := []struct {
		name  string
		event Event
		want  string
	}{
		{
			name: "full event",
			event: Event{
				Time: at, Provider: "github", Kind: "pr.commented", Repo: "dukex/foo", Number: 12,
				Actor: "alice", Text: "pode ver\no teste?", URL: "https://github.com/dukex/foo/pull/12#c1",
			},
			want: `2026-11-23T13:02:11Z github pr.commented dukex/foo#12 @alice "pode ver o teste?" https://github.com/dukex/foo/pull/12#c1`,
		},
		{
			name:  "watching thread",
			event: Event{Time: at, Provider: "github", Kind: "issue.closed", Repo: "a/b", Number: 3, Actor: "bob", Watching: true, URL: "u"},
			want:  `2026-11-23T13:02:11Z github issue.closed a/b#3 @bob watching u`,
		},
		{
			name:  "repo without a number",
			event: Event{Time: at, Provider: "github", Kind: "ci.failed", Repo: "a/b", Text: "CI failed"},
			want:  `2026-11-23T13:02:11Z github ci.failed a/b "CI failed"`,
		},
		{
			name:  "error event has no subject",
			event: Event{Time: at, Provider: "github", Kind: "error", Text: "boom"},
			want:  `2026-11-23T13:02:11Z github error "boom"`,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.event.String(); got != c.want {
				t.Errorf("\ngot  %s\nwant %s", got, c.want)
			}
		})
	}
}

func TestEventStringTruncatesLongText(t *testing.T) {
	e := Event{Provider: "p", Kind: "k", Text: strings.Repeat("é", 200)}
	got := e.String()
	if !strings.Contains(got, `"`+strings.Repeat("é", 140)+`…"`) {
		t.Errorf("text not truncated at 140 runes: %s", got)
	}
}
