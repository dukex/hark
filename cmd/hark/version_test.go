package main

import (
	"bytes"
	"context"
	"runtime/debug"
	"testing"
)

func TestVersionFlags(t *testing.T) {
	defer func(v string) { version = v }(version)
	version = "0.5.0"
	for _, flag := range []string{"--version", "-v", "version"} {
		var stdout, stderr bytes.Buffer
		stubUpdate(t, "0.5.0", func(string, string) (string, error) { t.Fatal("update checked"); return "", nil })
		if code := run(context.Background(), []string{flag}, env(nil), &stdout, &stderr); code != 0 {
			t.Errorf("%s: exit %d", flag, code)
		}
		if stdout.String() != "hark 0.5.0\n" || stderr.Len() != 0 {
			t.Errorf("%s: stdout %q stderr %q", flag, stdout.String(), stderr.String())
		}
	}
}

func TestDevBuildShowsModuleVersion(t *testing.T) {
	defer func(v string, b func() (*debug.BuildInfo, bool)) { version, buildInfo = v, b }(version, buildInfo)
	version = "dev"
	cases := map[string]string{"v0.5.0": "v0.5.0", "(devel)": "dev", "": "dev"}
	for module, want := range cases {
		buildInfo = func() (*debug.BuildInfo, bool) { return &debug.BuildInfo{Main: debug.Module{Version: module}}, true }
		if got := displayVersion(); got != want {
			t.Errorf("module %q: got %q, want %q", module, got, want)
		}
	}
	buildInfo = func() (*debug.BuildInfo, bool) { return nil, false }
	if got := displayVersion(); got != "dev" {
		t.Errorf("no build info: got %q", got)
	}
}
