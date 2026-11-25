package main

import (
	"cmp"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"runtime/debug"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/dukex/hark"
	"github.com/dukex/hark/datadog"
	"github.com/dukex/hark/discord"
	"github.com/dukex/hark/github"
	"github.com/dukex/hark/jira"
	"github.com/dukex/hark/linear"
	"github.com/dukex/hark/slack"
	"github.com/dukex/hark/todoist"
	"github.com/dukex/hark/update"
)

var (
	version    = "dev"
	buildInfo  = debug.ReadBuildInfo
	releases   = "https://github.com/dukex/hark/releases"
	restart    = syscall.Exec
	executable = os.Executable
	runUpdate  = func(ctx context.Context, exe, current string) (string, error) {
		return update.Updater{Releases: releases, Executable: exe}.Run(ctx, current)
	}
	exit    = os.Exit
	ghToken = func() (string, error) {
		out, err := exec.Command("gh", "auth", "token").Output()
		return strings.TrimSpace(string(out)), err
	}
)

var errNotConfigured = errors.New("not configured")

//go:embed help.txt
var helpText string

type options struct {
	Since  time.Time
	Filter string
}

type constructor func(getenv func(string) string, o options) (hark.Provider, error)

var providers = map[string]constructor{
	"datadog": newDatadog,
	"discord": newDiscord,
	"github":  newGitHub,
	"jira":    newJira,
	"linear":  newLinear,
	"slack":   newSlack,
	"todoist": newTodoist,
}

func newGitHub(getenv func(string) string, o options) (hark.Provider, error) {
	token := getenv("GITHUB_TOKEN")
	if token == "" {
		t, err := ghToken()
		if err != nil {
			return nil, fmt.Errorf("%w: set GITHUB_TOKEN or log in with gh: %w", errNotConfigured, err)
		}
		token = t
	}
	api := getenv("HARK_GITHUB_API")
	if api == "" {
		api = "https://api.github.com"
	}
	filter, err := github.ParseFilter(o.Filter)
	if err != nil {
		return nil, fmt.Errorf("--filter: %w", err)
	}
	w := github.New(api, token)
	w.Since, w.Filter = o.Since, filter
	return w, nil
}

func newTodoist(getenv func(string) string, o options) (hark.Provider, error) {
	token := getenv("TODOIST_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("%w: set TODOIST_API_TOKEN to a token from Todoist settings, Integrations, Developer", errNotConfigured)
	}
	state, err := hark.StatePath("todoist")
	if err != nil {
		return nil, err
	}
	w := todoist.New(cmp.Or(getenv("HARK_TODOIST_API"), "https://api.todoist.com/api/v1"), token, state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func newJira(getenv func(string) string, o options) (hark.Provider, error) {
	var missing []string
	for _, name := range []string{"JIRA_URL", "JIRA_EMAIL", "JIRA_API_TOKEN"} {
		if getenv(name) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: set %s (the token comes from id.atlassian.com, Security, API tokens)", errNotConfigured, strings.Join(missing, ", "))
	}
	state, err := hark.StatePath("jira")
	if err != nil {
		return nil, err
	}
	w := jira.New(getenv("JIRA_URL"), getenv("JIRA_EMAIL"), getenv("JIRA_API_TOKEN"), state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func newDatadog(getenv func(string) string, o options) (hark.Provider, error) {
	var missing []string
	for _, name := range []string{"DD_API_KEY", "DD_APP_KEY"} {
		if getenv(name) == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf("%w: set %s (keys from Datadog, Organization Settings, API Keys and Application Keys) and DD_SITE if not datadoghq.com", errNotConfigured, strings.Join(missing, ", "))
	}
	state, err := hark.StatePath("datadog")
	if err != nil {
		return nil, err
	}
	site := cmp.Or(getenv("DD_SITE"), "datadoghq.com")
	app := "https://" + site
	if strings.Count(site, ".") == 1 {
		app = "https://app." + site
	}
	w := datadog.New(cmp.Or(getenv("HARK_DATADOG_API"), "https://api."+site), app, getenv("DD_API_KEY"), getenv("DD_APP_KEY"), state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func newLinear(getenv func(string) string, o options) (hark.Provider, error) {
	key := getenv("LINEAR_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("%w: set LINEAR_API_KEY to a personal API key from Linear, Settings, Security & access", errNotConfigured)
	}
	state, err := hark.StatePath("linear")
	if err != nil {
		return nil, err
	}
	w := linear.New(cmp.Or(getenv("HARK_LINEAR_API"), "https://api.linear.app/graphql"), key, state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func newDiscord(getenv func(string) string, o options) (hark.Provider, error) {
	token, user := getenv("DISCORD_TOKEN"), getenv("DISCORD_USER_ID")
	if token == "" || user == "" {
		return nil, fmt.Errorf("%w: set DISCORD_TOKEN and DISCORD_USER_ID (the bot token of a Discord app with Message Content Intent, added to your servers, and your own user ID)", errNotConfigured)
	}
	state, err := hark.StatePath("discord")
	if err != nil {
		return nil, err
	}
	w := discord.New(cmp.Or(getenv("HARK_DISCORD_API"), "https://discord.com/api/v10"), token, user, state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func newSlack(getenv func(string) string, o options) (hark.Provider, error) {
	token := getenv("SLACK_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("%w: set SLACK_TOKEN to the user token (xoxp-) of a Slack app with the search:read user scope", errNotConfigured)
	}
	state, err := hark.StatePath("slack")
	if err != nil {
		return nil, err
	}
	w := slack.New(cmp.Or(getenv("HARK_SLACK_API"), "https://slack.com/api"), token, state)
	w.Since, w.Filter = o.Since, o.Filter
	return w, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	stop()
	exit(code)
}

func displayVersion() string {
	if info, ok := buildInfo(); ok && version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return version
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) == 0 || len(args) == 1 && slices.Contains([]string{"help", "--help", "-h"}, args[0]) {
		_, _ = fmt.Fprint(stdout, helpText)
		return 0
	}
	if len(args) == 1 && slices.Contains([]string{"--version", "-v", "version"}, args[0]) {
		_, _ = fmt.Fprintln(stdout, "hark", displayVersion())
		return 0
	}
	if len(args) == 1 && args[0] == "doctor" {
		return doctor(ctx, getenv, stdout)
	}
	if len(args) == 1 && args[0] == "update" {
		return selfUpdate(ctx, stdout, stderr)
	}
	if providers[args[0]] == nil {
		return usage(stderr)
	}
	var o options
	flags := flag.NewFlagSet("hark "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.Usage = func() {}
	flags.Func("since", "", func(s string) (err error) {
		o.Since, err = parseSince(s)
		return err
	})
	flags.StringVar(&o.Filter, "filter", "", "")
	exitOnEvent := flags.Bool("exit-on-event", false, "")
	err := flags.Parse(args[1:])
	if errors.Is(err, flag.ErrHelp) {
		_, _ = fmt.Fprint(stdout, helpText)
		return 0
	}
	if err != nil || flags.NArg() > 0 {
		return usage(stderr)
	}
	autoUpdate(ctx, args, getenv, stderr)
	p, err := providers[args[0]](getenv, o)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "hark:", err)
		return 1
	}
	watch, stop := context.WithCancel(ctx)
	defer stop()
	show := func(e hark.Event) {
		_, _ = fmt.Fprintln(stdout, e)
		if *exitOnEvent && e.Kind != "error" {
			stop()
		}
	}
	if err := p.Watch(watch, onceUntilRecovered(show)); err != nil {
		_, _ = fmt.Fprintln(stderr, "hark:", err)
		return 1
	}
	return 0
}

func usage(stderr io.Writer) int {
	_, _ = fmt.Fprintf(stderr, "usage: hark <%s> [--since TIME] [--filter RULES] [--exit-on-event]\nrun `hark help` for the full reference\n", strings.Join(slices.Sorted(maps.Keys(providers)), "|"))
	return 2
}

func parseSince(s string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t, nil
	}
	if t, err := time.ParseInLocation(time.DateOnly, s, time.Local); err == nil {
		return t, nil
	}
	return time.Time{}, errors.New("want a time like 2026-11-23T10:05:00Z or a date like 2026-11-23")
}

func autoUpdate(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) {
	if version == "dev" || getenv("HARK_NO_UPDATE") != "" {
		return
	}
	exe, tag, err := install(ctx, version)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "hark: update failed:", err)
		return
	}
	if tag == "" {
		return
	}
	_, _ = fmt.Fprintf(stderr, "hark: updated %s to %s, restarting\n", version, tag)
	err = restart(exe, append([]string{exe}, args...), os.Environ())
	_, _ = fmt.Fprintln(stderr, "hark: restart failed:", err)
}

func doctor(ctx context.Context, getenv func(string) string, stdout io.Writer) int {
	_, _ = fmt.Fprintln(stdout, "hark", displayVersion())
	failed, configured := false, 0
	for _, name := range slices.Sorted(maps.Keys(providers)) {
		_, _ = fmt.Fprintln(stdout, name)
		p, err := providers[name](getenv, options{})
		if errors.Is(err, errNotConfigured) {
			_, _ = fmt.Fprintln(stdout, "  -", err)
			continue
		}
		if err != nil {
			_, _ = fmt.Fprintln(stdout, "  ✗ setup:", err)
			failed = true
			continue
		}
		configured++
		for _, c := range p.Doctor(ctx) {
			if c.Err != nil {
				_, _ = fmt.Fprintf(stdout, "  ✗ %s: %v\n", c.Name, c.Err)
				failed = true
			} else {
				_, _ = fmt.Fprintf(stdout, "  ✓ %s: %s\n", c.Name, c.Detail)
			}
		}
	}
	if configured == 0 {
		_, _ = fmt.Fprintln(stdout, "no provider is configured")
		failed = true
	}
	if failed {
		return 1
	}
	return 0
}

func install(ctx context.Context, current string) (exe, tag string, err error) {
	if exe, err = executable(); err != nil {
		return "", "", err
	}
	tag, err = runUpdate(ctx, exe, current)
	return exe, tag, err
}

func selfUpdate(ctx context.Context, stdout, stderr io.Writer) int {
	current := displayVersion()
	_, tag, err := install(ctx, current)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "hark: update failed:", err)
		return 1
	}
	if tag == "" {
		_, _ = fmt.Fprintf(stdout, "hark %s is up to date\n", current)
		return 0
	}
	_, _ = fmt.Fprintf(stdout, "hark updated %s to %s\n", current, tag)
	return 0
}

func onceUntilRecovered(emit func(hark.Event)) func(hark.Event) {
	failing := map[string]bool{}
	return func(e hark.Event) {
		if e.Kind != "error" {
			clear(failing)
		} else if failing[e.Text] {
			return
		}
		failing[e.Text] = e.Kind == "error"
		emit(e)
	}
}
