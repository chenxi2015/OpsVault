package selfupdate

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"1.0.0", "v1.0.0"},
		{"v1.0.0", "v1.0.0"},
		{" v2.3.4 ", "v2.3.4"},
		{"dev", "dev"},
		{"latest", "latest"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeTag(tc.in); got != tc.want {
			t.Errorf("NormalizeTag(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestIsSameVersion(t *testing.T) {
	if !IsSameVersion("v1.2.0", "1.2.0") {
		t.Fatal("expected v1.2.0 and 1.2.0 to be considered same version")
	}
	if IsSameVersion("dev", "v1.2.0") {
		t.Fatal("expected dev and v1.2.0 not to be same version")
	}
	if IsSameVersion("v1.1.0", "v1.2.0") {
		t.Fatal("expected v1.1.0 and v1.2.0 not to be same version")
	}
}

func TestExtractTagFromLocation(t *testing.T) {
	loc := "https://github.com/chenxi2015/opsvault/releases/tag/v1.3.2"
	if got := extractTagFromLocation(loc); got != "v1.3.2" {
		t.Fatalf("expected v1.3.2, got %q", got)
	}
}

func TestVerifyFileChecksum(t *testing.T) {
	tmpDir := t.TempDir()
	binFile := filepath.Join(tmpDir, "opsvault-linux-amd64")
	payload := []byte("mock-binary-payload-v1.0.0")
	if err := os.WriteFile(binFile, payload, 0o755); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(payload)
	hexSum := hex.EncodeToString(sum[:])

	checksumsTxt := fmt.Sprintf("%s  opsvault-linux-amd64\nabcdef123456  opsvault-darwin-arm64\n", hexSum)

	if err := VerifyFileChecksum(binFile, "opsvault-linux-amd64", checksumsTxt); err != nil {
		t.Fatalf("expected valid checksum, got error: %v", err)
	}

	badChecksumsTxt := "0000000000000000000000000000000000000000000000000000000000000000  opsvault-linux-amd64\n"
	if err := VerifyFileChecksum(binFile, "opsvault-linux-amd64", badChecksumsTxt); err == nil {
		t.Fatal("expected checksum mismatch error, got nil")
	}
}

func TestAtomicReplaceBinaryAndSymlink(t *testing.T) {
	tmpDir := t.TempDir()
	realBin := filepath.Join(tmpDir, "opsvault-real")
	symlinkBin := filepath.Join(tmpDir, "opsvault-link")
	newBin := filepath.Join(tmpDir, "opsvault.new")

	if err := os.WriteFile(realBin, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realBin, symlinkBin); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("new-binary-v2"), 0o755); err != nil {
		t.Fatal(err)
	}

	u := New(Options{ExecutablePath: symlinkBin})
	resolved, err := u.resolveExecutablePath()
	if err != nil {
		t.Fatal(err)
	}

	if err := AtomicReplaceBinary(resolved, newBin); err != nil {
		t.Fatalf("AtomicReplaceBinary failed: %v", err)
	}

	data, err := os.ReadFile(symlinkBin)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new-binary-v2" {
		t.Fatalf("expected updated binary content 'new-binary-v2', got %q", string(data))
	}
}

func TestUpdaterRunWithExplicitTagAndMockServer(t *testing.T) {
	tmpDir := t.TempDir()
	currentBin := filepath.Join(tmpDir, "opsvault")
	if err := os.WriteFile(currentBin, []byte("old-v0.9.0"), 0o755); err != nil {
		t.Fatal(err)
	}

	newPayload := []byte("new-binary-v1.0.0")
	sum := sha256.Sum256(newPayload)
	hexSum := hex.EncodeToString(sum[:])

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/https://github.com/chenxi2015/opsvault/releases/download/v1.0.0/opsvault-linux-amd64":
			_, _ = w.Write(newPayload)
		case "/https://github.com/chenxi2015/opsvault/releases/download/v1.0.0/checksums.txt":
			_, _ = fmt.Fprintf(w, "%s  opsvault-linux-amd64\n", hexSum)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	updater := New(Options{
		Repo:           "chenxi2015/opsvault",
		CurrentVersion: "v0.9.0",
		TargetTag:      "1.0.0",
		OS:             "linux",
		Arch:           "amd64",
		Proxies:        []string{srv.URL + "/"},
		ExecutablePath: currentBin,
		HTTPClient:     srv.Client(),
	})

	res, err := updater.Run()
	if err != nil {
		t.Fatalf("updater.Run() unexpected error: %v", err)
	}
	if !res.Updated || res.TargetVersion != "v1.0.0" {
		t.Fatalf("unexpected result: %+v", res)
	}

	gotBytes, err := os.ReadFile(currentBin)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotBytes) != string(newPayload) {
		t.Fatalf("expected binary to be updated to %q, got %q", string(newPayload), string(gotBytes))
	}
}

