package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/dukex/hark"
)

const (
	backlog   = time.Hour
	lag       = 5 * time.Minute
	epoch     = 1420070400000
	limit     = 25
	maxOffset = 9975
	intents   = 1<<18 | 1<<19
	userAgent = "DiscordBot (https://github.com/dukex/hark, 1)"
)

var errNoServer = errors.New("the bot is in no server: add it to the servers to watch")

type Watcher struct {
	API      string
	Token    string
	UserID   string
	State    string
	Interval time.Duration
	Now      func() time.Time
	Since    time.Time
	Filter   string

	state state
}

type state struct {
	Guilds map[string]cursor `json:"guilds"`
}

type cursor struct {
	Cursor time.Time `json:"cursor"`
	Seen   []string  `json:"seen"`
}

func New(api, token, userID, statePath string) *Watcher {
	return &Watcher{
		API:      strings.TrimSuffix(api, "/"),
		Token:    token,
		UserID:   userID,
		State:    statePath,
		Interval: time.Minute,
		Now:      time.Now,
	}
}

type user struct {
	ID       string `json:"id"`
	Username string `json:"username"`
}

type guild struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type message struct {
	ID        string    `json:"id"`
	ChannelID string    `json:"channel_id"`
	Author    user      `json:"author"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type results struct {
	TotalResults int         `json:"total_results"`
	Messages     [][]message `json:"messages"`
}

func (w *Watcher) Watch(ctx context.Context, emit func(hark.Event)) error {
	var bot, me user
	if err := w.get(ctx, "/users/@me", nil, &bot); err != nil {
		return err
	}
	if err := w.get(ctx, "/users/"+w.UserID, nil, &me); err != nil {
		return fmt.Errorf("DISCORD_USER_ID %s: %w", w.UserID, err)
	}
	guilds, err := w.guilds(ctx)
	if err != nil {
		return err
	}
	if w.Filter != "" {
		if _, err := url.ParseQuery(w.Filter); err != nil {
			return fmt.Errorf("--filter %q: %w", w.Filter, err)
		}
		q := w.query("mentions")
		q.Set("limit", "1")
		var probe results
		if err := w.get(ctx, "/guilds/"+guilds[0].ID+"/messages/search", q, &probe); err != nil {
			return fmt.Errorf("--filter %q: %w", w.Filter, err)
		}
	}
	if err := hark.LoadState(w.State, &w.state); err != nil {
		return err
	}
	if !w.Since.IsZero() {
		w.state.Guilds = nil
	}
	for {
		for _, e := range w.poll(ctx) {
			emit(e)
		}
		if err := hark.SaveState(w.State, w.state); err != nil {
			emit(w.failure(err))
		}
		select {
		case <-ctx.Done():
			return nil
		case <-time.After(w.Interval):
		}
	}
}

func (w *Watcher) guilds(ctx context.Context) ([]guild, error) {
	var guilds []guild
	if err := w.get(ctx, "/users/@me/guilds", nil, &guilds); err != nil {
		return nil, err
	}
	if len(guilds) == 0 {
		return nil, errNoServer
	}
	return guilds, nil
}

func (w *Watcher) query(by string) url.Values {
	q := url.Values{by: {w.UserID}, "sort_by": {"timestamp"}, "sort_order": {"desc"}, "limit": {strconv.Itoa(limit)}}
	extra, _ := url.ParseQuery(w.Filter)
	for k, vs := range extra {
		q[k] = append(q[k], vs...)
	}
	return q
}

func (w *Watcher) poll(ctx context.Context) []hark.Event {
	var guilds []guild
	if err := w.get(ctx, "/users/@me/guilds", nil, &guilds); err != nil {
		return []hark.Event{w.failure(err)}
	}
	start := w.Since
	if start.IsZero() {
		start = w.Now().Add(-backlog)
	}
	next := map[string]cursor{}
	var events []hark.Event
	for _, g := range guilds {
		c, ok := w.state.Guilds[g.ID]
		if !ok {
			c.Cursor = start
		}
		found, moved, err := w.pollGuild(ctx, g, c)
		if err != nil {
			if ok {
				next[g.ID] = c
			}
			events = append(events, w.failure(fmt.Errorf("%s: %w", g.Name, err)))
			continue
		}
		next[g.ID] = moved
		events = append(events, found...)
	}
	slices.SortStableFunc(events, func(a, b hark.Event) int { return a.Time.Compare(b.Time) })
	w.state.Guilds = next
	return events
}

func (w *Watcher) pollGuild(ctx context.Context, g guild, c cursor) ([]hark.Event, cursor, error) {
	next := w.Now().Add(-lag)
	if next.Before(c.Cursor) {
		next = c.Cursor
	}
	found := map[string]bool{}
	var events []hark.Event
	var seen []string
	for _, by := range []string{"mentions", "replied_to_user_id"} {
		q := w.query(by)
		q.Set("min_id", snowflake(c.Cursor))
		total := 1
		for offset := 0; offset < total && offset <= maxOffset; offset += limit {
			q.Set("offset", strconv.Itoa(offset))
			var res results
			if err := w.get(ctx, "/guilds/"+g.ID+"/messages/search", q, &res); err != nil {
				return nil, c, err
			}
			total = res.TotalResults
			for _, hit := range res.Messages {
				for _, m := range hit {
					if m.Timestamp.Before(c.Cursor) || found[m.ID] {
						continue
					}
					found[m.ID] = true
					if !m.Timestamp.Before(next) {
						seen = append(seen, m.ID)
					}
					if m.Author.ID == w.UserID || slices.Contains(c.Seen, m.ID) {
						continue
					}
					kind := "discord.mentioned"
					if by != "mentions" {
						kind = "discord.replied"
					}
					events = append(events, hark.Event{
						Time:     m.Timestamp.UTC(),
						Provider: "discord",
						Kind:     kind,
						Actor:    m.Author.Username,
						Text:     g.Name + ": " + m.Content,
						URL:      "https://discord.com/channels/" + g.ID + "/" + m.ChannelID + "/" + m.ID,
					})
				}
			}
		}
	}
	return events, cursor{Cursor: next, Seen: seen}, nil
}

func snowflake(t time.Time) string {
	return strconv.FormatInt((t.UnixMilli()-epoch)<<22, 10)
}

func (w *Watcher) failure(err error) hark.Event {
	return hark.Event{Time: w.Now(), Provider: "discord", Kind: "error", Text: err.Error()}
}

func (w *Watcher) get(ctx context.Context, path string, q url.Values, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, w.API+path+"?"+q.Encode(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bot "+w.Token)
	req.Header.Set("User-Agent", userAgent)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode == http.StatusOK {
		return json.Unmarshal(body, v)
	}
	var e struct {
		Message string `json:"message"`
	}
	_ = json.Unmarshal(body, &e)
	switch {
	case res.StatusCode == http.StatusAccepted:
		return fmt.Errorf("GET %s: %s", path, e.Message)
	case e.Message != "":
		return fmt.Errorf("GET %s: %s: %s", path, res.Status, e.Message)
	}
	return fmt.Errorf("GET %s: %s", path, res.Status)
}

func (w *Watcher) Doctor(ctx context.Context) []hark.Check {
	var bot, me user
	if err := w.get(ctx, "/users/@me", nil, &bot); err != nil {
		return []hark.Check{{Name: "token", Err: err}}
	}
	var app struct {
		Flags int `json:"flags"`
	}
	intent := w.get(ctx, "/applications/@me", nil, &app)
	if intent == nil && app.Flags&intents == 0 {
		intent = errors.New("turn on Message Content Intent on the Bot page of the app")
	}
	userErr := w.get(ctx, "/users/"+w.UserID, nil, &me)
	guilds, err := w.guilds(ctx)
	var names []string
	var mentions int
	var searchErr error
	for _, g := range guilds {
		names = append(names, g.Name)
		q := url.Values{"mentions": {w.UserID}, "min_id": {snowflake(w.Now().AddDate(0, 0, -7))}, "limit": {"1"}}
		var res results
		if err := w.get(ctx, "/guilds/"+g.ID+"/messages/search", q, &res); err != nil && searchErr == nil {
			searchErr = fmt.Errorf("%s: %w", g.Name, err)
		}
		mentions += res.TotalResults
	}
	return []hark.Check{
		{Name: "token", Detail: "logged in as " + bot.Username},
		{Name: "message content", Detail: "intent on", Err: intent},
		{Name: "user", Detail: "watching mentions of " + me.Username, Err: userErr},
		{Name: "servers", Detail: strings.Join(names, ", "), Err: err},
		{Name: "search", Detail: fmt.Sprintf("%d mentions in the last 7 days", mentions), Err: searchErr},
		hark.CheckState(w.State),
	}
}
