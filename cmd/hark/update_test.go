package main

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
)

func stubUpdate(t *testing.T, current string, result func(exe, current string) (string, error)) *[]string {
	t.Helper()
	oldVersion, oldRun, oldRestart, oldExe := version, runUpdate, restart, executable
	t.Cleanup(func() { version, runUpdate, restart, executable = oldVersion, oldRun, oldRestart, oldExe })
	version = current
	executable = func() (string, error) { return "/opt/hark", nil }
	var restarted []string
	restart = func(exe string, argv, env []string) error {
		restarted = append([]string{exe}, argv...)
		return errors.New("exec stubbed")
	}
	runUpdate = func(_ context.Context, exe, current string) (string, error) { return result(exe, current) }
	return &restarted
}

func TestNewerReleaseRestartsHark(t *testing.T) {
	var asked string
	restarted := stubUpdate(t, "0.1.0", func(exe, current string) (string, error) {
		asked = exe + " " + current
		return "v0.2.0", nil
	})
	var stderr bytes.Buffer
	autoUpdate(context.Background(), []string{"github"}, env(nil), &stderr)

	if asked != "/opt/hark 0.1.0" {
		t.Errorf("updater got %q", asked)
	}
	if strings.Join(*restarted, " ") != "/opt/hark /opt/hark github" {
		t.Errorf("restarted %v", *restarted)
	}
	if stderr.String() != "hark: updated 0.1.0 to v0.2.0, restarting\nhark: restart failed: exec stubbed\n" {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestUpToDateKeepsRunning(t *testing.T) {
	restarted := stubUpdate(t, "0.2.0", func(string, string) (string, error) { return "", nil })
	var stderr bytes.Buffer
	autoUpdate(context.Background(), []string{"github"}, env(nil), &stderr)
	if len(*restarted) != 0 || stderr.Len() != 0 {
		t.Errorf("restarted %v, stderr %q", *restarted, stderr.String())
	}
}

func TestUpdateFailureIsReportedAndIgnored(t *testing.T) {
	stubUpdate(t, "0.1.0", func(string, string) (string, error) { return "", errors.New("offline") })
	var stderr bytes.Buffer
	autoUpdate(context.Background(), nil, env(nil), &stderr)
	if stderr.String() != "hark: update failed: offline\n" {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestMissingExecutableIsReported(t *testing.T) {
	stubUpdate(t, "0.1.0", func(string, string) (string, error) { t.Fatal("updater called"); return "", nil })
	executable = func() (string, error) { return "", errors.New("no path") }
	var stderr bytes.Buffer
	autoUpdate(context.Background(), nil, env(nil), &stderr)
	if stderr.String() != "hark: update failed: no path\n" {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestUpdateIsSkippedForDevBuildsAndOptOut(t *testing.T) {
	cases := map[string]struct {
		version string
		env     map[string]string
	}{
		"dev build": {"dev", nil},
		"opt out":   {"0.1.0", map[string]string{"HARK_NO_UPDATE": "1"}},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			stubUpdate(t, c.version, func(string, string) (string, error) { t.Fatal("updater called"); return "", nil })
			autoUpdate(context.Background(), nil, env(c.env), &bytes.Buffer{})
		})
	}
}

func TestRealUpdaterReportsUnreachableReleases(t *testing.T) {
	defer func(r string) { releases = r }(releases)
	releases = "http://127.0.0.1:1"
	if _, err := runUpdate(context.Background(), "/nonexistent/hark", "0.1.0"); err == nil {
		t.Fatal("want error")
	}
}

func TestRunChecksForUpdatesFirst(t *testing.T) {
	stubUpdate(t, "0.1.0", func(string, string) (string, error) { return "", errors.New("offline") })
	var stderr bytes.Buffer
	run(context.Background(), []string{"github"}, env(map[string]string{"GITHUB_TOKEN": "t", "HARK_GITHUB_API": "http://127.0.0.1:1"}), nil, &stderr)
	if !strings.HasPrefix(stderr.String(), "hark: update failed: offline\n") {
		t.Errorf("stderr %q", stderr.String())
	}
}
