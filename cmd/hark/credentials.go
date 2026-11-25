package main

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

const service = "hark"

var (
	goos            = runtime.GOOS
	stdin io.Reader = os.Stdin
)

type field struct {
	Name     string
	About    string
	Secret   bool
	Optional bool
}

var logins = map[string][]field{
	"datadog": {
		{Name: "DD_API_KEY", About: "Organization Settings, API Keys", Secret: true},
		{Name: "DD_APP_KEY", About: "Organization Settings, Application Keys, with events_read", Secret: true},
		{Name: "DD_SITE", About: "like datadoghq.eu, empty for datadoghq.com", Optional: true},
	},
	"discord": {
		{Name: "DISCORD_TOKEN", About: "the token on the Bot page of your Discord app", Secret: true},
		{Name: "DISCORD_USER_ID", About: "your own user ID: Developer Mode on, right-click your name, Copy User ID"},
	},
	"github": {
		{Name: "GITHUB_TOKEN", About: "empty to keep using `gh auth token`", Secret: true, Optional: true},
	},
	"jira": {
		{Name: "JIRA_URL", About: "your Jira Cloud site, like https://example.atlassian.net"},
		{Name: "JIRA_EMAIL", About: "the e-mail of your Atlassian account"},
		{Name: "JIRA_API_TOKEN", About: "id.atlassian.com, Security, API tokens", Secret: true},
	},
	"linear": {
		{Name: "LINEAR_API_KEY", About: "Linear, Settings, Security & access, Personal API keys", Secret: true},
	},
	"slack": {
		{Name: "SLACK_TOKEN", About: "the user token (xoxp-) of a Slack app with the search:read user scope", Secret: true},
	},
	"todoist": {
		{Name: "TODOIST_API_TOKEN", About: "Todoist settings, Integrations, Developer", Secret: true},
	},
}

func credentials(getenv func(string) string) func(string) string {
	known := map[string]bool{}
	for _, fields := range logins {
		for _, f := range fields {
			known[f.Name] = true
		}
	}
	file := readCredentials(credentialsPath(getenv))
	return func(name string) string {
		if !known[name] {
			return getenv(name)
		}
		if v := keychainGet(getenv, name); v != "" {
			return v
		}
		return cmp.Or(file[name], getenv(name))
	}
}

func command(getenv func(string) string, name string, args ...string) *exec.Cmd {
	for _, dir := range filepath.SplitList(getenv("PATH")) {
		path := filepath.Join(dir, name)
		if info, err := os.Stat(path); err == nil && info.Mode()&0o111 != 0 {
			return exec.CommandContext(context.Background(), path, args...)
		}
	}
	return nil
}

func keychainCommand(getenv func(string) string, args ...string) *exec.Cmd {
	switch goos {
	case "darwin":
		return command(getenv, "security", args...)
	case "linux":
		return command(getenv, "secret-tool", args...)
	}
	return nil
}

func keychainGet(getenv func(string) string, name string) string {
	args := []string{"lookup", "service", service, "account", name}
	if goos == "darwin" {
		args = []string{"find-generic-password", "-s", service, "-a", name, "-w"}
	}
	cmd := keychainCommand(getenv, args...)
	if cmd == nil {
		return ""
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(out), "\n")
}

var quote = strings.NewReplacer(`\`, `\\`, `"`, `\"`)

func keychainSet(getenv func(string) string, name, value string) bool {
	cmd := keychainCommand(getenv, "store", "--label="+service+" "+name, "service", service, "account", name)
	input := value
	if goos == "darwin" {
		cmd = keychainCommand(getenv, "-i")
		input = fmt.Sprintf("add-generic-password -U -s %s -a %s -w \"%s\"\n", service, name, quote.Replace(value))
	}
	if cmd == nil {
		return false
	}
	cmd.Stdin = strings.NewReader(input)
	return cmd.Run() == nil && keychainGet(getenv, name) == value
}

func credentialsPath(getenv func(string) string) string {
	config := getenv("XDG_CONFIG_HOME")
	if config == "" && getenv("HOME") != "" {
		config = filepath.Join(getenv("HOME"), ".config")
	}
	if config == "" {
		return ""
	}
	return filepath.Join(config, "hark", "credentials")
}

func readCredentials(path string) map[string]string {
	values := map[string]string{}
	b, _ := os.ReadFile(path)
	for line := range strings.Lines(string(b)) {
		if name, value, ok := strings.Cut(strings.TrimSpace(line), "="); ok && !strings.HasPrefix(name, "#") {
			values[name] = value
		}
	}
	return values
}

func writeCredentials(path string, values map[string]string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	saved := readCredentials(path)
	maps.Copy(saved, values)
	var b strings.Builder
	for _, name := range slices.Sorted(maps.Keys(saved)) {
		fmt.Fprintf(&b, "%s=%s\n", name, saved[name])
	}
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func echo(getenv func(string) string, on bool) {
	f, ok := stdin.(*os.File)
	arg := "-echo"
	if on {
		arg = "echo"
	}
	if cmd := command(getenv, "stty", arg); ok && cmd != nil {
		cmd.Stdin = f
		_ = cmd.Run()
	}
}

func login(getenv func(string) string, provider string, stdout, stderr io.Writer) int {
	fields, ok := logins[provider]
	if !ok {
		_, _ = fmt.Fprintf(stderr, "usage: hark login <%s>\n", strings.Join(slices.Sorted(maps.Keys(logins)), "|"))
		return 2
	}
	in := bufio.NewReader(stdin)
	values := map[string]string{}
	for _, f := range fields {
		_, _ = fmt.Fprintf(stdout, "%s (%s): ", f.Name, f.About)
		if f.Secret {
			echo(getenv, false)
		}
		line, err := in.ReadString('\n')
		if f.Secret {
			echo(getenv, true)
			_, _ = fmt.Fprintln(stdout)
		}
		value := strings.TrimSpace(line)
		switch {
		case err != nil && value == "" && !f.Optional:
			_, _ = fmt.Fprintf(stderr, "hark: %s: %v\n", f.Name, err)
			return 1
		case value == "" && !f.Optional:
			_, _ = fmt.Fprintf(stderr, "hark: %s is required\n", f.Name)
			return 1
		case value != "":
			values[f.Name] = value
		}
	}
	where, err := save(getenv, values)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, "hark:", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%s saved to %s. Run `hark doctor` to check it.\n", provider, where)
	return 0
}

func save(getenv func(string) string, values map[string]string) (string, error) {
	kept := true
	for _, name := range slices.Sorted(maps.Keys(values)) {
		kept = kept && keychainSet(getenv, name, values[name])
	}
	if kept && goos == "darwin" {
		return "the macOS Keychain", nil
	}
	if kept && goos == "linux" {
		return "the Secret Service keyring", nil
	}
	path := credentialsPath(getenv)
	if path == "" {
		return "", errors.New("no keychain, and no HOME or XDG_CONFIG_HOME for ~/.config/hark/credentials")
	}
	if err := writeCredentials(path, values); err != nil {
		return "", err
	}
	return path + ", readable only by you", nil
}
