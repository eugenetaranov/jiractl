package selfupdate

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewer(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"0.3.1", "v0.3.2", true},
		{"v0.3.2", "v0.3.2", false},
		{"0.10.0", "v0.9.9", false},
		{"0.9.9", "v0.10.0", true},
		{"1.0.0", "v0.99.0", false},
		{"dev", "v0.3.2", true},
		{"v0.3.1-2-gabc-dirty", "v0.3.2", true},
		{"0.3.2", "garbage", false},
	}
	for _, c := range cases {
		if got := Newer(c.current, c.latest); got != c.want {
			t.Errorf("Newer(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
	if !IsDev("dev") || IsDev("0.3.2") || IsDev("v0.3.2") {
		t.Error("IsDev")
	}
}

func TestIsBrew(t *testing.T) {
	if !IsBrew("/opt/homebrew/Cellar/jiractl/0.3.2/bin/jiractl") {
		t.Error("Cellar path not detected")
	}
	if IsBrew("/usr/local/bin/jiractl") || IsBrew("/Users/x/go/bin/jiractl") {
		t.Error("non-brew path detected as brew")
	}
}

func tarGz(t *testing.T, name string, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range []struct {
		name string
		body []byte
	}{{"README.md", []byte("readme")}, {name, body}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.body)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(f.body); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func releaseServer(t *testing.T, archive []byte, checksum string) *httptest.Server {
	t.Helper()
	name := archiveName("v0.4.0")
	mux := http.NewServeMux()
	mux.HandleFunc("/repos/"+Repo+"/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tag_name":"v0.4.0"}`))
	})
	dl := "/" + Repo + "/releases/download/v0.4.0/"
	mux.HandleFunc(dl+"checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("deadbeef  jiractl_0.4.0_other_arch.tar.gz\n" + checksum + "  " + name + "\n"))
	})
	mux.HandleFunc(dl+name, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func TestLatestAndInstall(t *testing.T) {
	archive := tarGz(t, "jiractl", []byte("new binary"))
	sum := sha256.Sum256(archive)
	srv := releaseServer(t, archive, hex.EncodeToString(sum[:]))
	u := &Updater{APIBase: srv.URL, DownloadBase: srv.URL}

	tag, err := u.Latest(context.Background())
	if err != nil || tag != "v0.4.0" {
		t.Fatalf("Latest = %q, %v", tag, err)
	}

	exe := filepath.Join(t.TempDir(), "jiractl")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := u.Install(context.Background(), tag, exe); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new binary" {
		t.Fatalf("binary = %q", got)
	}
	if fi, _ := os.Stat(exe); fi.Mode().Perm() != 0o755 {
		t.Fatalf("mode %v", fi.Mode())
	}
	entries, _ := os.ReadDir(filepath.Dir(exe))
	if len(entries) != 1 {
		t.Fatalf("temp file left behind: %v", entries)
	}
}

func TestInstallRejectsBadChecksum(t *testing.T) {
	archive := tarGz(t, "jiractl", []byte("tampered"))
	srv := releaseServer(t, archive, strings.Repeat("0", 64))
	u := &Updater{APIBase: srv.URL, DownloadBase: srv.URL}

	exe := filepath.Join(t.TempDir(), "jiractl")
	if err := os.WriteFile(exe, []byte("old binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := u.Install(context.Background(), "v0.4.0", exe)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("err = %v", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old binary" {
		t.Fatalf("binary replaced despite bad checksum: %q", got)
	}
}
