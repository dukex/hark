package main

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"os/signal"
	"slices"
	"strings"
	"syscall"

	"github.com/dukex/hark"
	"github.com/dukex/hark/github"
	"github.com/dukex/hark/todoist"
	"github.com/dukex/hark/update"
)

var (
	version    = "dev"
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

var providers = map[string]func(getenv func(string) string) (hark.Provider, error){
	"github":  newGitHub,
	"todoist": newTodoist,
}

func newGitHub(getenv func(string) string) (hark.Provider, error) {
	token := getenv("GITHUB_TOKEN")
	if token == "" {
		t, err := ghToken()
		if err != nil {
			return nil, fmt.Errorf("set GITHUB_TOKEN or log in with gh: %w", err)
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
		return nil, fmt.Errorf("set TODOIST_API_TOKEN to a token from Todoist settings, Integrations, Developer")
	}
	state, err := hark.StatePath("todoist")
	if err != nil {
		return nil, err
	}
	return todoist.New(cmp.Or(getenv("HARK_TODOIST_API"), "https://api.todoist.com/api/v1"), token, state), nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	code := run(ctx, os.Args[1:], os.Getenv, os.Stdout, os.Stderr)
	stop()
	exit(code)
}

func run(ctx context.Context, args []string, getenv func(string) string, stdout, stderr io.Writer) int {
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
