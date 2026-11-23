package main

import (
	"cmp"
	"context"
	"errors"
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

	"github.com/dukex/hark"
	"github.com/dukex/hark/github"
	"github.com/dukex/hark/jira"
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

var providers = map[string]func(getenv func(string) string) (hark.Provider, error){
	"github":  newGitHub,
	"jira":    newJira,
	"todoist": newTodoist,
}

func newGitHub(getenv func(string) string) (hark.Provider, error) {
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
	return github.New(api, token), nil
}

func newTodoist(getenv func(string) string) (hark.Provider, error) {
	token := getenv("TODOIST_API_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("%w: set TODOIST_API_TOKEN to a token from Todoist settings, Integrations, Developer", errNotConfigured)
	}
	state, err := hark.StatePath("todoist")
	if err != nil {
		return nil, err
	}
	return todoist.New(cmp.Or(getenv("HARK_TODOIST_API"), "https://api.todoist.com/api/v1"), token, state), nil
}

func newJira(getenv func(string) string) (hark.Provider, error) {
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
	return jira.New(getenv("JIRA_URL"), getenv("JIRA_EMAIL"), getenv("JIRA_API_TOKEN"), state), nil
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
	if len(args) == 1 && slices.Contains([]string{"--version", "-v", "version"}, args[0]) {
		_, _ = fmt.Fprintln(stdout, "hark", displayVersion())
		return 0
	}
	if len(args) == 1 && args[0] == "doctor" {
		return doctor(ctx, getenv, stdout)
	}
	if len(args) != 1 || providers[args[0]] == nil {
		_, _ = fmt.Fprintf(stderr, "usage: hark <%s>\n", strings.Join(slices.Sorted(maps.Keys(providers)), "|"))
		return 2
	}
	autoUpdate(ctx, args, getenv, stderr)
	p, err := providers[args[0]](getenv)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "hark:", err)
		return 1
	}
	if err := p.Watch(ctx, func(e hark.Event) { _, _ = fmt.Fprintln(stdout, e) }); err != nil {
		_, _ = fmt.Fprintln(stderr, "hark:", err)
		return 1
	}
	return 0
}

func autoUpdate(ctx context.Context, args []string, getenv func(string) string, stderr io.Writer) {
	if version == "dev" || getenv("HARK_NO_UPDATE") != "" {
		return
	}
	exe, err := executable()
	tag := ""
	if err == nil {
		tag, err = runUpdate(ctx, exe, version)
	}
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
		p, err := providers[name](getenv)
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
