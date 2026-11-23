package main

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestUpdateCommand(t *testing.T) {
	cases := map[string]struct {
		result      func(exe, current string) (string, error)
		code        int
		out, errOut string
	}{
		"newer release": {func(string, string) (string, error) { return "v0.7.0", nil }, 0, "hark updated 0.6.0 to v0.7.0\n", ""},
		"up to date":    {func(string, string) (string, error) { return "", nil }, 0, "hark 0.6.0 is up to date\n", ""},
		"failure":       {func(string, string) (string, error) { return "", errors.New("offline") }, 1, "", "hark: update failed: offline\n"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			restarted := stubUpdate(t, "0.6.0", c.result)
			var stdout, stderr bytes.Buffer
			code := run(context.Background(), []string{"update"}, env(map[string]string{"HARK_NO_UPDATE": "1"}), &stdout, &stderr)
			if code != c.code || stdout.String() != c.out || stderr.String() != c.errOut || len(*restarted) != 0 {
				t.Errorf("exit %d stdout %q stderr %q restarted %v", code, stdout.String(), stderr.String(), *restarted)
			}
		})
	}
}

func TestUpdateCommandWithoutExecutable(t *testing.T) {
	stubUpdate(t, "0.6.0", func(string, string) (string, error) { t.Fatal("updater called"); return "", nil })
	executable = func() (string, error) { return "", errors.New("no path") }
	var stderr bytes.Buffer
	if code := run(context.Background(), []string{"update"}, env(nil), &bytes.Buffer{}, &stderr); code != 1 || stderr.String() != "hark: update failed: no path\n" {
		t.Errorf("exit %d stderr %q", code, stderr.String())
	}
}
