package main

import (
	"bytes"
	"context"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

const fakeSecurity = `#!/bin/sh
echo "$@" >> "$KEYCHAIN/calls"
if [ "$1" = "-i" ]; then
	read -r line
	eval "set -- $line"
	while [ $# -gt 0 ]; do
		case "$1" in
		-a) account=$2; shift ;;
		-w) password=$2; shift ;;
		esac
		shift
	done
	[ -n "$BROKEN" ] && exit 0
	printf '%s' "$password" > "$KEYCHAIN/$account"
	exit 0
fi
while [ $# -gt 0 ]; do
	[ "$1" = "-a" ] && account=$2
	shift
done
[ -f "$KEYCHAIN/$account" ] || exit 44
cat "$KEYCHAIN/$account"
echo
`

const fakeSecretTool = `#!/bin/sh
echo "$@" >> "$KEYCHAIN/calls"
cmd=$1
while [ $# -gt 0 ]; do
	[ "$1" = "account" ] && account=$2
	shift
done
if [ "$cmd" = "store" ]; then
	cat > "$KEYCHAIN/$account"
	exit 0
fi
[ -f "$KEYCHAIN/$account" ] || exit 1
cat "$KEYCHAIN/$account"
`

type keychain struct {
	dir, bin string
}

func fakeKeychain(t *testing.T, system string) keychain {
	t.Helper()
	k := keychain{dir: t.TempDir(), bin: t.TempDir()}
	t.Setenv("KEYCHAIN", k.dir)
	t.Setenv("BROKEN", "")
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(k.bin, name), []byte(body), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	write("security", fakeSecurity)
	write("secret-tool", fakeSecretTool)
	write("stty", "#!/bin/sh\necho \"stty $*\" >> \"$KEYCHAIN/calls\"\n")
	old := goos
	goos = system
	t.Cleanup(func() { goos = old })
	return k
}

func (k keychain) set(t *testing.T, name, value string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(k.dir, name), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (k keychain) get(name string) string {
	b, _ := os.ReadFile(filepath.Join(k.dir, name))
	return string(b)
}

func (k keychain) calls() string {
	b, _ := os.ReadFile(filepath.Join(k.dir, "calls"))
	return string(b)
}

func withStdin(t *testing.T, r *os.File) {
	t.Helper()
	old := stdin
	stdin = r
	t.Cleanup(func() { stdin = old })
}

func answers(t *testing.T, lines ...string) {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdin")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(strings.Join(lines, "\n"))
	_, _ = f.Seek(0, 0)
	withStdin(t, f)
}

func TestCredentialsComeFromKeychainThenFileThenEnv(t *testing.T) {
	for _, system := range []string{"darwin", "linux"} {
		t.Run(system, func(t *testing.T) {
			k := fakeKeychain(t, system)
			k.set(t, "SLACK_TOKEN", "from-keychain")
			config := t.TempDir()
			saveFile(t, config, "SLACK_TOKEN=from-file\nLINEAR_API_KEY=from-file\n# comment\nnot a pair\n")
			getenv := credentials(env(map[string]string{
				"PATH":              k.bin,
				"XDG_CONFIG_HOME":   config,
				"SLACK_TOKEN":       "from-env",
				"LINEAR_API_KEY":    "from-env",
				"TODOIST_API_TOKEN": "from-env",
				"HARK_NO_UPDATE":    "1",
			}))
			for name, want := range map[string]string{
				"SLACK_TOKEN":       "from-keychain",
				"LINEAR_API_KEY":    "from-file",
				"TODOIST_API_TOKEN": "from-env",
				"HARK_NO_UPDATE":    "1",
				"JIRA_URL":          "",
			} {
				if got := getenv(name); got != want {
					t.Errorf("%s = %q, want %q", name, got, want)
				}
			}
			if strings.Contains(k.calls(), "HARK_NO_UPDATE") {
				t.Error("looked up a setting that is not a credential in the keychain")
			}
		})
	}
}

func TestCredentialsWithoutKeychainOrFileComeFromEnv(t *testing.T) {
	fakeKeychain(t, "windows")
	getenv := credentials(env(map[string]string{"SLACK_TOKEN": "from-env", "HOME": t.TempDir()}))
	if got := getenv("SLACK_TOKEN"); got != "from-env" {
		t.Errorf("got %q", got)
	}
	if got := credentials(env(map[string]string{"SLACK_TOKEN": "from-env"}))("SLACK_TOKEN"); got != "from-env" {
		t.Errorf("without HOME: got %q", got)
	}
}

func saveFile(t *testing.T, config, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(config, "hark"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(config, "hark", "credentials"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestEveryProviderCanLogIn(t *testing.T) {
	if got, want := slices.Sorted(maps.Keys(logins)), slices.Sorted(maps.Keys(providers)); !slices.Equal(got, want) {
		t.Errorf("logins %v, providers %v", got, want)
	}
}

func TestLoginSavesToTheMacOSKeychain(t *testing.T) {
	k := fakeKeychain(t, "darwin")
	answers(t, "https://example.atlassian.net", "duke@example.com", `tok"en\x`)
	var stdout, stderr bytes.Buffer
	code := run(context.Background(), []string{"login", "jira"}, env(map[string]string{"PATH": k.bin}), &stdout, &stderr)

	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
	for name, want := range map[string]string{"JIRA_URL": "https://example.atlassian.net", "JIRA_EMAIL": "duke@example.com", "JIRA_API_TOKEN": `tok"en\x`} {
		if got := k.get(name); got != want {
			t.Errorf("%s = %q, want %q", name, got, want)
		}
	}
	out := stdout.String()
	if !strings.Contains(out, "JIRA_URL (your Jira Cloud site, like https://example.atlassian.net): ") || !strings.HasSuffix(out, "saved to the macOS Keychain. Run `hark doctor` to check it.\n") {
		t.Errorf("stdout %q", out)
	}
	if strings.Contains(out, `tok"en`) || strings.Contains(k.calls(), `tok"en`) && !strings.Contains(k.calls(), "-i") {
		t.Error("the token was echoed or passed as an argument")
	}
	if !strings.Contains(k.calls(), "stty -echo") || !strings.Contains(k.calls(), "stty echo") {
		t.Errorf("echo not turned off for the token: %q", k.calls())
	}
}

func TestLoginSavesToTheLinuxSecretService(t *testing.T) {
	k := fakeKeychain(t, "linux")
	answers(t, "xoxp-1")
	var stdout bytes.Buffer
	if code := run(context.Background(), []string{"login", "slack"}, env(map[string]string{"PATH": k.bin}), &stdout, nil); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if k.get("SLACK_TOKEN") != "xoxp-1" || !strings.HasSuffix(stdout.String(), "saved to the Secret Service keyring. Run `hark doctor` to check it.\n") {
		t.Errorf("keychain %q, stdout %q", k.get("SLACK_TOKEN"), stdout.String())
	}
	if !strings.Contains(k.calls(), "store --label=hark SLACK_TOKEN service hark account SLACK_TOKEN") {
		t.Errorf("calls %q", k.calls())
	}
}

func TestLoginFallsBackToAFile(t *testing.T) {
	for name, setup := range map[string]func(t *testing.T) map[string]string{
		"no keychain": func(t *testing.T) map[string]string {
			fakeKeychain(t, "darwin")
			return map[string]string{"PATH": t.TempDir()}
		},
		"keychain that does not keep it": func(t *testing.T) map[string]string {
			k := fakeKeychain(t, "darwin")
			t.Setenv("BROKEN", "1")
			return map[string]string{"PATH": k.bin}
		},
	} {
		t.Run(name, func(t *testing.T) {
			vars := setup(t)
			home := t.TempDir()
			vars["HOME"] = home
			saveFile(t, filepath.Join(home, ".config"), "TODOIST_API_TOKEN=old\nSLACK_TOKEN=xoxp-1\n")
			answers(t, "new")
			var stdout bytes.Buffer
			if code := run(context.Background(), []string{"login", "todoist"}, env(vars), &stdout, nil); code != 0 {
				t.Fatalf("exit %d", code)
			}
			path := filepath.Join(home, ".config", "hark", "credentials")
			b, _ := os.ReadFile(path)
			if string(b) != "SLACK_TOKEN=xoxp-1\nTODOIST_API_TOKEN=new\n" {
				t.Errorf("file %q", b)
			}
			if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
				t.Errorf("mode %v", info.Mode())
			}
			if !strings.HasSuffix(stdout.String(), "saved to "+path+", readable only by you. Run `hark doctor` to check it.\n") {
				t.Errorf("stdout %q", stdout.String())
			}
		})
	}
}

func TestLoginSkipsOptionalAnswers(t *testing.T) {
	k := fakeKeychain(t, "darwin")
	answers(t, "api", "app", "")
	if code := run(context.Background(), []string{"login", "datadog"}, env(map[string]string{"PATH": k.bin}), &bytes.Buffer{}, nil); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if k.get("DD_API_KEY") != "api" || k.get("DD_APP_KEY") != "app" || k.get("DD_SITE") != "" {
		t.Errorf("saved %q %q %q", k.get("DD_API_KEY"), k.get("DD_APP_KEY"), k.get("DD_SITE"))
	}
}

func TestLoginFailures(t *testing.T) {
	k := fakeKeychain(t, "darwin")
	for _, c := range []struct {
		args   []string
		vars   map[string]string
		lines  []string
		code   int
		stderr string
	}{
		{[]string{"login"}, nil, nil, 2, "usage: hark login <datadog|discord|github|jira|linear|slack|todoist>\n"},
		{[]string{"login", "gitlab"}, nil, nil, 2, "usage: hark login <datadog|discord|github|jira|linear|slack|todoist>\n"},
		{[]string{"login", "jira"}, map[string]string{"PATH": k.bin}, []string{"https://example.atlassian.net", "", "token"}, 1, "hark: JIRA_EMAIL is required\n"},
		{[]string{"login", "linear"}, map[string]string{"PATH": k.bin}, nil, 1, "hark: LINEAR_API_KEY: EOF\n"},
		{[]string{"login", "linear"}, nil, []string{"lin_api_1"}, 1, "hark: no keychain, and no HOME or XDG_CONFIG_HOME for ~/.config/hark/credentials\n"},
	} {
		answers(t, c.lines...)
		var stderr bytes.Buffer
		if code := run(context.Background(), c.args, env(c.vars), &bytes.Buffer{}, &stderr); code != c.code || stderr.String() != c.stderr {
			t.Errorf("%v: exit %d, stderr %q", c.args, code, stderr.String())
		}
	}

	config := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(config, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	answers(t, "lin_api_1")
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"login", "linear"}, env(map[string]string{"XDG_CONFIG_HOME": config}), &bytes.Buffer{}, &stderr); code != 1 || !strings.HasPrefix(stderr.String(), "hark: mkdir ") {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}

	taken := t.TempDir()
	if err := os.MkdirAll(filepath.Join(taken, "hark", "credentials"), 0o700); err != nil {
		t.Fatal(err)
	}
	answers(t, "lin_api_1")
	stderr.Reset()
	if code := run(context.Background(), []string{"login", "linear"}, env(map[string]string{"XDG_CONFIG_HOME": taken}), &bytes.Buffer{}, &stderr); code != 1 || !strings.HasSuffix(stderr.String(), "is a directory\n") {
		t.Errorf("exit %d, stderr %q", code, stderr.String())
	}
}

func TestLoginThenDoctor(t *testing.T) {
	defer func(f func() (string, error)) { ghToken = f }(ghToken)
	ghToken = func() (string, error) { return "", errors.New("not logged in") }
	k := fakeKeychain(t, "darwin")
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	answers(t, "lin_api_1")
	vars := map[string]string{"PATH": k.bin, "HARK_LINEAR_API": "http://127.0.0.1:1"}
	if code := run(context.Background(), []string{"login", "linear"}, env(vars), &bytes.Buffer{}, nil); code != 0 {
		t.Fatalf("login exit %d", code)
	}
	var stdout bytes.Buffer
	run(context.Background(), []string{"doctor"}, env(vars), &stdout, nil)
	if !strings.Contains(stdout.String(), "\nlinear\n  ✗ api key: Post \"http://127.0.0.1:1\"") {
		t.Errorf("doctor did not use the key from the keychain:\n%s", stdout.String())
	}
}
