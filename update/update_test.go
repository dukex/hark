package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

var asset = fmt.Sprintf("hark_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)

func tarball(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type release struct {
	tag       string
	archive   []byte
	checksums string
}

func goodRelease(t *testing.T, tag string) release {
	archive := tarball(t, "hark", []byte("new binary"))
	return release{tag: tag, archive: archive, checksums: fmt.Sprintf("%x  %s\n", sha256.Sum256(archive), asset)}
}

func serve(t *testing.T, r release) string {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/latest":
			if r.tag != "" {
				http.Redirect(w, req, "/tag/"+r.tag, http.StatusFound)
				return
			}
			w.WriteHeader(http.StatusOK)
		case "/download/" + r.tag + "/" + asset:
			writeOr404(w, req, r.archive)
		case "/download/" + r.tag + "/checksums.txt":
			writeOr404(w, req, []byte(r.checksums))
		default:
			http.NotFound(w, req)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func writeOr404(w http.ResponseWriter, req *http.Request, body []byte) {
	if len(body) == 0 {
		http.NotFound(w, req)
		return
	}
	_, _ = w.Write(body)
}

func executable(t *testing.T) string {
	exe := filepath.Join(t.TempDir(), "hark")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	return exe
}

func content(t *testing.T, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestReplacesBinaryWhenReleaseIsNewer(t *testing.T) {
	exe := executable(t)
	u := Updater{Releases: serve(t, goodRelease(t, "v0.10.0")), Executable: exe}

	tag, err := u.Run(context.Background(), "0.9.1")
	if err != nil || tag != "v0.10.0" {
		t.Fatalf("tag %q, err %v", tag, err)
	}
	if got := content(t, exe); got != "new binary" {
		t.Errorf("executable holds %q", got)
	}
	if info, _ := os.Stat(exe); info.Mode().Perm() != 0o755 {
		t.Errorf("mode %v", info.Mode())
	}
}

func TestKeepsBinaryWhenUpToDate(t *testing.T) {
	for _, current := range []string{"0.10.0", "v0.10.0", "0.11.0", "1.0.0", "dev", "0.10", "1.x.0"} {
		exe := executable(t)
		u := Updater{Releases: serve(t, goodRelease(t, "v0.10.0")), Executable: exe}
		if tag, err := u.Run(context.Background(), current); err != nil || tag != "" {
			t.Errorf("%s: tag %q, err %v", current, tag, err)
		}
		if got := content(t, exe); got != "old binary" {
			t.Errorf("%s: executable replaced", current)
		}
	}
}

func TestFailuresKeepTheOldBinary(t *testing.T) {
	bad := func(edit func(*release)) release {
		r := goodRelease(t, "v0.2.0")
		edit(&r)
		return r
	}
	cases := map[string]struct {
		releases string
		want     string
	}{
		"unreachable":       {"http://127.0.0.1:1", "connection refused"},
		"bad url":           {"http://bad host", "invalid"},
		"no redirect":       {serve(t, release{}), "no release found"},
		"bad tag":           {serve(t, bad(func(r *release) { r.tag = "latest-build" })), ""},
		"asset missing":     {serve(t, bad(func(r *release) { r.archive = nil; r.checksums = "" })), "404"},
		"not in checksums":  {serve(t, bad(func(r *release) { r.checksums = "abc  other.tar.gz\n" })), "no checksum for"},
		"checksum mismatch": {serve(t, bad(func(r *release) { r.checksums = "00  " + asset + "\n" })), "checksum mismatch"},
		"not gzip": {serve(t, bad(func(r *release) {
			r.archive = []byte("plain text, not gzip")
			r.checksums = fmt.Sprintf("%x  %s\n", sha256.Sum256(r.archive), asset)
		})), "gzip"},
		"no binary in archive": {serve(t, bad(func(r *release) {
			r.archive = tarball(t, "README.md", []byte("hi"))
			r.checksums = fmt.Sprintf("%x  %s\n", sha256.Sum256(r.archive), asset)
		})), "no hark binary"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			exe := executable(t)
			tag, err := Updater{Releases: c.releases, Executable: exe}.Run(context.Background(), "0.1.0")
			if c.want == "" {
				if err != nil || tag != "" {
					t.Fatalf("tag %q, err %v", tag, err)
				}
			} else if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("err %v, want %q", err, c.want)
			}
			if got := content(t, exe); got != "old binary" {
				t.Errorf("executable replaced: %q", got)
			}
		})
	}
}

func TestReadOnlyDirectoryKeepsTheOldBinary(t *testing.T) {
	exe := executable(t)
	dir := filepath.Dir(exe)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })

	_, err := Updater{Releases: serve(t, goodRelease(t, "v0.2.0")), Executable: exe}.Run(context.Background(), "0.1.0")
	if err == nil {
		t.Fatal("want error writing next to a read-only executable")
	}
	if got := content(t, exe); got != "old binary" {
		t.Errorf("executable replaced: %q", got)
	}
}

func TestCorruptTarKeepsTheOldBinary(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write([]byte("not a tar"))
	_ = gz.Close()
	r := release{tag: "v0.2.0", archive: buf.Bytes(), checksums: fmt.Sprintf("%x  %s\n", sha256.Sum256(buf.Bytes()), asset)}
	exe := executable(t)
	if _, err := (Updater{Releases: serve(t, r), Executable: exe}).Run(context.Background(), "0.1.0"); err == nil {
		t.Fatal("want tar error")
	}
}

func TestRenameFailureCleansUp(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "hark")
	if err := os.MkdirAll(filepath.Join(exe, "busy"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := (Updater{Releases: serve(t, goodRelease(t, "v0.2.0")), Executable: exe}).Run(context.Background(), "0.1.0"); err == nil {
		t.Fatal("want rename error")
	}
	if _, err := os.Stat(exe + ".new"); !os.IsNotExist(err) {
		t.Errorf("leftover %s.new", exe)
	}
}

func TestChecksumsMissing(t *testing.T) {
	r := goodRelease(t, "v0.2.0")
	r.checksums = ""
	if _, err := (Updater{Releases: serve(t, r), Executable: executable(t)}).Run(context.Background(), "0.1.0"); err == nil || !strings.Contains(err.Error(), "checksums.txt: 404") {
		t.Fatalf("err %v", err)
	}
}

func TestDroppedDownloadKeepsTheOldBinary(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/latest" {
			http.Redirect(w, req, "/tag/v0.2.0", http.StatusFound)
			return
		}
		conn, _, _ := w.(http.Hijacker).Hijack()
		_ = conn.Close()
	}))
	defer srv.Close()
	exe := executable(t)
	if _, err := (Updater{Releases: srv.URL, Executable: exe}).Run(context.Background(), "0.1.0"); err == nil {
		t.Fatal("want download error")
	}
	if got := content(t, exe); got != "old binary" {
		t.Errorf("executable replaced: %q", got)
	}
}
