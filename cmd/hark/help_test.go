package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestHelpPrintsTheReference(t *testing.T) {
	stubUpdate(t, "0.5.0", func(string, string) (string, error) { t.Fatal("update checked"); return "", nil })
	for _, args := range [][]string{nil, {"help"}, {"--help"}, {"-h"}, {"github", "--help"}, {"todoist", "-h"}} {
		var stdout, stderr bytes.Buffer
		if code := run(context.Background(), args, env(nil), &stdout, &stderr); code != 0 {
			t.Errorf("%v: exit %d", args, code)
		}
		if stdout.String() != helpText || stderr.Len() != 0 {
			t.Errorf("%v: stdout %d bytes, stderr %q", args, stdout.Len(), stderr.String())
		}
	}
}

func TestHelpCoversEveryProviderAndFlag(t *testing.T) {
	for name := range providers {
		if !strings.Contains(helpText, "\n  "+name) {
			t.Errorf("help has no section for %s", name)
		}
	}
	for _, s := range []string{"--since TIME", "--filter RULES", "hark doctor", "hark update", "HARK_NO_UPDATE", "EXIT STATUS"} {
		if !strings.Contains(helpText, s) {
			t.Errorf("help does not mention %s", s)
		}
	}
}
