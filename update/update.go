package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
)

// Updater replaces Executable with the latest release published under Releases
// (for example https://github.com/dukex/hark/releases) when it is newer.
type Updater struct {
	Releases   string
	Executable string
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

// Run returns the tag it installed, or "" when current is already the latest.
func (u Updater) Run(ctx context.Context, current string) (string, error) {
	tag, err := u.latest(ctx)
	if err != nil {
		return "", err
	}
	latest, ok := parse(tag)
	running, ok2 := parse(current)
	if !ok || !ok2 || slices.Compare(latest, running) <= 0 {
		return "", nil
	}
	asset := fmt.Sprintf("hark_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	archive, err := u.download(ctx, tag, asset)
	if err != nil {
		return "", err
	}
	sums, err := u.download(ctx, tag, "checksums.txt")
	if err != nil {
		return "", err
	}
	if err := verify(archive, sums, asset); err != nil {
		return "", err
	}
	bin, err := extract(archive)
	if err != nil {
		return "", err
	}
	return tag, replace(u.Executable, bin)
}

func (u Updater) latest(ctx context.Context) (string, error) {
	res, err := get(ctx, noRedirect, u.Releases+"/latest")
	if err != nil {
		return "", err
	}
	defer func() { _ = res.Body.Close() }()
	location := res.Header.Get("Location")
	if location == "" {
		return "", errors.New("no release found")
	}
	return path.Base(location), nil
}

func (u Updater) download(ctx context.Context, tag, name string) ([]byte, error) {
	res, err := get(ctx, http.DefaultClient, u.Releases+"/download/"+tag+"/"+name)
	if err != nil {
		return nil, err
	}
	defer func() { _ = res.Body.Close() }()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download %s: %s", name, res.Status)
	}
	return io.ReadAll(res.Body)
}

func get(ctx context.Context, client *http.Client, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	return client.Do(req)
}

func parse(version string) ([]int, bool) {
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	if len(parts) != 3 {
		return nil, false
	}
	nums := make([]int, 3)
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, false
		}
		nums[i] = n
	}
	return nums, true
}

func verify(archive, sums []byte, asset string) error {
	for line := range strings.Lines(string(sums)) {
		if f := strings.Fields(line); len(f) == 2 && f[1] == asset {
			if f[0] != fmt.Sprintf("%x", sha256.Sum256(archive)) {
				return fmt.Errorf("checksum mismatch for %s", asset)
			}
			return nil
		}
	}
	return fmt.Errorf("no checksum for %s", asset)
}

func extract(archive []byte) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil, errors.New("no hark binary in archive")
		}
		if err != nil {
			return nil, err
		}
		if hdr.Name == "hark" {
			return io.ReadAll(tr)
		}
	}
}

func replace(exe string, bin []byte) error {
	next := exe + ".new"
	if err := os.WriteFile(next, bin, 0o755); err != nil {
		return err
	}
	if err := os.Rename(next, exe); err != nil {
		_ = os.Remove(next)
		return err
	}
	return nil
}
