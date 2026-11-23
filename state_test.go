package hark

import (
	"os"
	"path/filepath"
	"testing"
)

type cursor struct {
	Token string
	Seen  []string
}

func TestStatePathFollowsXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/xdg")
	if got, err := StatePath("jira"); err != nil || got != "/xdg/hark/jira.json" {
		t.Errorf("got %q, %v", got, err)
	}
	t.Setenv("XDG_STATE_HOME", "")
	t.Setenv("HOME", "/home/me")
	if got, err := StatePath("jira"); err != nil || got != "/home/me/.local/state/hark/jira.json" {
		t.Errorf("got %q, %v", got, err)
	}
	t.Setenv("HOME", "")
	if _, err := StatePath("jira"); err == nil {
		t.Error("want error without HOME")
	}
}

func TestStateRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "todoist.json")
	var empty cursor
	if err := LoadState(path, &empty); err != nil || empty.Token != "" {
		t.Fatalf("missing file: %+v, %v", empty, err)
	}
	if err := SaveState(path, cursor{Token: "abc", Seen: []string{"x"}}); err != nil {
		t.Fatal(err)
	}
	var got cursor
	if err := LoadState(path, &got); err != nil || got.Token != "abc" || len(got.Seen) != 1 {
		t.Fatalf("got %+v, %v", got, err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Error("temporary file left behind")
	}
}

func TestStateErrors(t *testing.T) {
	dir := t.TempDir()
	if err := LoadState(dir, &cursor{}); err == nil {
		t.Error("loading a directory should fail")
	}
	bad := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(bad, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadState(bad, &cursor{}); err == nil {
		t.Error("loading invalid JSON should fail")
	}
	if err := SaveState(filepath.Join(bad, "x.json"), cursor{}); err == nil {
		t.Error("saving under a file should fail")
	}
	if err := SaveState(filepath.Join(dir, "chan.json"), make(chan int)); err == nil {
		t.Error("saving an unencodable value should fail")
	}
	if err := SaveState(dir, cursor{}); err == nil {
		t.Error("renaming over a directory should fail")
	}
	readOnly := filepath.Join(dir, "ro")
	if err := os.Mkdir(readOnly, 0o555); err != nil {
		t.Fatal(err)
	}
	if err := SaveState(filepath.Join(readOnly, "x.json"), cursor{}); err == nil {
		t.Error("saving in a read-only directory should fail")
	}
}

func TestCheckState(t *testing.T) {
	dir := t.TempDir()
	if c := CheckState(filepath.Join(dir, "missing.json")); c.Name != "state" || c.Err != nil || c.Detail != filepath.Join(dir, "missing.json") {
		t.Errorf("missing file: %+v", c)
	}
	if c := CheckState(dir); c.Err == nil {
		t.Errorf("directory as state: %+v", c)
	}
}
