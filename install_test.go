package hark

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func fakeRelease(t *testing.T, checksum func(sum [32]byte) [32]byte) string {
	t.Helper()
	var archive bytes.Buffer
	gz := gzip.NewWriter(&archive)
	tw := tar.NewWriter(gz)
	bin := []byte("#!/bin/sh\necho fake-hark\n")
	if err := tw.WriteHeader(&tar.Header{Name: "hark", Mode: 0o755, Size: int64(len(bin))}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(bin); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	dir := t.TempDir()
	asset := fmt.Sprintf("hark_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	sum := checksum(sha256.Sum256(archive.Bytes()))
	files := map[string][]byte{
		asset:           archive.Bytes(),
		"checksums.txt": fmt.Appendf(nil, "%x  hark_other.tar.gz\n%x  %s\n", [32]byte{}, sum, asset),
	}
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	srv := httptest.NewServer(http.FileServer(http.Dir(dir)))
	t.Cleanup(srv.Close)
	return srv.URL
}

func install(t *testing.T, url string) (string, []byte, error) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "bin")
	cmd := exec.Command("sh", "install.sh")
	cmd.Env = append(os.Environ(), "HARK_DOWNLOAD_URL="+url, "HARK_INSTALL_DIR="+dir)
	out, err := cmd.CombinedOutput()
	return filepath.Join(dir, "hark"), out, err
}

func TestInstallScriptInstallsReleaseBinary(t *testing.T) {
	bin, out, err := install(t, fakeRelease(t, func(s [32]byte) [32]byte { return s }))
	if err != nil {
		t.Fatalf("install.sh failed: %v\n%s", err, out)
	}
	got, err := exec.Command(bin).Output()
	if err != nil || string(got) != "fake-hark\n" {
		t.Fatalf("installed binary: %q, %v\ninstall output:\n%s", got, err, out)
	}
}

func TestInstallScriptRejectsBadChecksum(t *testing.T) {
	bin, out, err := install(t, fakeRelease(t, func([32]byte) [32]byte { return [32]byte{1} }))
	if err == nil {
		t.Fatalf("install.sh accepted a bad checksum:\n%s", out)
	}
	if _, statErr := os.Stat(bin); !os.IsNotExist(statErr) {
		t.Fatalf("binary installed despite bad checksum")
	}
}
