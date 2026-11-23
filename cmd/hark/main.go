package main

import (
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
)

var (
	exit    = os.Exit
	ghToken = func() (string, error) {
		out, err := exec.Command("gh", "auth", "token").Output()
		return strings.TrimSpace(string(out)), err
	}
)

var providers = map[string]func(getenv func(string) string) (hark.Provider, error){
	"github": newGitHub,
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
