// Package selfupdate finds the latest jiractl release and installs it, with
// Homebrew when jiractl came from the tap, or by replacing the running
// binary with the release archive otherwise.
package selfupdate

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	Repo    = "eugenetaranov/jiractl"
	Tap     = "eugenetaranov/tap"
	Formula = "jiractl"
	Binary  = "jiractl"
)

// Updater talks to GitHub. The zero value uses github.com.
type Updater struct {
	// APIBase and DownloadBase default to https://api.github.com and
	// https://github.com (tests point them at a local server).
	APIBase      string
	DownloadBase string
	HTTP         *http.Client
}

func (u *Updater) apiBase() string {
	if u.APIBase != "" {
		return u.APIBase
	}
	return "https://api.github.com"
}

func (u *Updater) downloadBase() string {
	if u.DownloadBase != "" {
		return u.DownloadBase
	}
	return "https://github.com"
}

func (u *Updater) client() *http.Client {
	if u.HTTP != nil {
		return u.HTTP
	}
	return &http.Client{Timeout: 2 * time.Minute}
}

func (u *Updater) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := u.client().Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("GET %s: %s", url, resp.Status)
	}
	return resp, nil
}

// Latest returns the tag of the latest published release, e.g. "v0.3.2".
func (u *Updater) Latest(ctx context.Context) (string, error) {
	resp, err := u.get(ctx, u.apiBase()+"/repos/"+Repo+"/releases/latest")
	if err != nil {
		return "", fmt.Errorf("could not check the latest release: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("could not read the latest release: %w", err)
	}
	if rel.TagName == "" {
		return "", errors.New("the latest release has no tag")
	}
	return rel.TagName, nil
}

// IsDev reports whether version is not a release build (e.g. "dev" from
// go build, or "v0.3.1-2-gabc123-dirty" from git describe).
func IsDev(version string) bool {
	_, ok := parse(version)
	return !ok
}

// Newer reports whether latest is a later release than current. A version
// that isn't vX.Y.Z is never considered up to date.
func Newer(current, latest string) bool {
	c, ok := parse(current)
	if !ok {
		return true
	}
	l, ok := parse(latest)
	if !ok {
		return false
	}
	for i := range c {
		if l[i] != c[i] {
			return l[i] > c[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	var out [3]int
	parts := strings.Split(strings.TrimPrefix(v, "v"), ".")
	if len(parts) != 3 {
		return out, false
	}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 {
			return out, false
		}
		out[i] = n
	}
	return out, true
}

// Executable returns the running binary's real path (symlinks resolved).
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// IsBrew reports whether exe lives in a Homebrew Cellar.
func IsBrew(exe string) bool {
	return strings.Contains(filepath.ToSlash(exe), "/Cellar/"+Formula+"/")
}

// archiveName is the release archive for this platform, as .goreleaser.yaml
// names it.
func archiveName(tag string) string {
	return fmt.Sprintf("%s_%s_%s_%s.tar.gz", Binary, strings.TrimPrefix(tag, "v"), runtime.GOOS, runtime.GOARCH)
}

// Install downloads the tag's archive for this platform, checks it against
// the release's checksums.txt and replaces the binary at exe with it.
func (u *Updater) Install(ctx context.Context, tag, exe string) error {
	base := u.downloadBase() + "/" + Repo + "/releases/download/" + tag + "/"
	name := archiveName(tag)

	want, err := u.checksum(ctx, base+"checksums.txt", name)
	if err != nil {
		return err
	}

	resp, err := u.get(ctx, base+name)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", name, err)
	}
	defer func() { _ = resp.Body.Close() }()
	archive, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("could not download %s: %w", name, err)
	}
	sum := sha256.Sum256(archive)
	if got := hex.EncodeToString(sum[:]); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", name, got, want)
	}

	bin, err := extract(archive, Binary)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return replace(exe, bin)
}

func (u *Updater) checksum(ctx context.Context, url, name string) (string, error) {
	resp, err := u.get(ctx, url)
	if err != nil {
		return "", fmt.Errorf("could not download checksums: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	sc := bufio.NewScanner(resp.Body)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 2 && fields[1] == name {
			return fields[0], nil
		}
	}
	if err := sc.Err(); err != nil {
		return "", fmt.Errorf("could not read checksums: %w", err)
	}
	return "", fmt.Errorf("no release archive for %s/%s (%s)", runtime.GOOS, runtime.GOARCH, name)
}

// extract returns the file called name from a .tar.gz.
func extract(archive []byte, name string) ([]byte, error) {
	gz, err := gzip.NewReader(bytes.NewReader(archive))
	if err != nil {
		return nil, err
	}
	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			return nil, fmt.Errorf("archive has no %s", name)
		}
		if err != nil {
			return nil, err
		}
		if h.Typeflag == tar.TypeReg && filepath.Base(h.Name) == name {
			return io.ReadAll(tr)
		}
	}
}

// replace swaps exe for bin: it writes next to exe and renames over it, so
// a failure leaves the old binary in place.
func replace(exe string, bin []byte) error {
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, "."+Binary+"-upgrade-*")
	if err != nil {
		if errors.Is(err, os.ErrPermission) {
			return fmt.Errorf("can't write to %s: %w (rerun with sudo)", dir, err)
		}
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	if _, err := tmp.Write(bin); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		return fmt.Errorf("could not replace %s: %w", exe, err)
	}
	return nil
}
