package updater

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// --- Task 3.1: binaryName tests ---

func TestBinaryName(t *testing.T) {
	tests := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"windows", "amd64", "gtx-windows-amd64.exe"},
		{"linux", "amd64", "gtx-linux-amd64"},
		{"darwin", "arm64", "gtx-darwin-arm64"},
		{"darwin", "amd64", "gtx-darwin-amd64"},
		{"freebsd", "amd64", ""},
		{"linux", "arm64", ""},
	}
	for _, tc := range tests {
		key := tc.goos + "/" + tc.goarch
		got := platformBinary[key]
		if got != tc.want {
			t.Errorf("platformBinary[%q] = %q, want %q", key, got, tc.want)
		}
	}
}

func TestBinaryNameCurrentPlatform(t *testing.T) {
	name := binaryName()
	key := runtime.GOOS + "/" + runtime.GOARCH
	if _, supported := platformBinary[key]; !supported {
		if name != "" {
			t.Errorf("unsupported platform %s should return empty, got %q", key, name)
		}
		t.Skipf("platform %s not in supported list — binaryName returns empty as expected", key)
	}
	if name == "" {
		t.Errorf("supported platform %s returned empty binary name", key)
	}
}

// --- Task 3.2: LatestVersion / PrintVersion tests ---

func TestLatestVersion_NewVersionAvailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{TagName: "v0.3.0"})
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	tag, err := LatestVersion(gtxRepo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v0.3.0" {
		t.Errorf("got %q, want %q", tag, "v0.3.0")
	}
}

func TestLatestVersion_SameVersion(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{TagName: "v0.2.0"})
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	tag, err := LatestVersion(gtxRepo)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tag != "v0.2.0" {
		t.Errorf("got %q, want %q", tag, "v0.2.0")
	}
}

func TestLatestVersion_Timeout(t *testing.T) {
	// Server that never responds
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until client disconnects
		<-r.Context().Done()
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	_, err := LatestVersion(gtxRepo)
	if err == nil {
		t.Error("expected timeout error, got nil")
	}
}

func TestLatestVersion_HTTP403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	_, err := LatestVersion(gtxRepo)
	if err == nil {
		t.Error("expected error for HTTP 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention 403, got: %v", err)
	}
}

// --- Task 3.3: SelfUpdate integration test ---

func TestSelfUpdate_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		// On Windows, renaming the temp file while tests are running is unreliable
		// in the test harness — skip the atomic rename part.
		t.Skip("skipping SelfUpdate rename test on Windows")
	}

	const latestTag = "v0.9.9"
	const fakeContent = "#!/bin/sh\necho fake binary\n"

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{TagName: latestTag})
	}))
	defer apiSrv.Close()

	dlSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, fakeContent)
	}))
	defer dlSrv.Close()

	oldAPI := apiBaseURL
	oldDL := downloadBaseURL
	apiBaseURL = apiSrv.URL
	downloadBaseURL = dlSrv.URL
	defer func() {
		apiBaseURL = oldAPI
		downloadBaseURL = oldDL
	}()

	// Create a temp "executable" to be replaced
	dir := t.TempDir()
	fakeBin := filepath.Join(dir, "gtx")
	if err := os.WriteFile(fakeBin, []byte("old binary"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := selfUpdate("v0.1.0", fakeBin); err != nil {
		t.Fatalf("selfUpdate failed: %v", err)
	}

	got, err := os.ReadFile(fakeBin)
	if err != nil {
		t.Fatalf("reading updated binary: %v", err)
	}
	if string(got) != fakeContent {
		t.Errorf("binary content = %q, want %q", string(got), fakeContent)
	}

	// .old file should be cleaned up
	if _, err := os.Stat(fakeBin + ".old"); !os.IsNotExist(err) {
		t.Error("expected .old backup to be removed after successful update")
	}
}

func TestSelfUpdate_AlreadyLatest(t *testing.T) {
	const currentVersion = "v0.2.0"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(releaseResponse{TagName: currentVersion})
	}))
	defer srv.Close()

	old := apiBaseURL
	apiBaseURL = srv.URL
	defer func() { apiBaseURL = old }()

	err := selfUpdate(currentVersion, "/nonexistent/path/gtx")
	if err != nil {
		t.Errorf("expected no error when already up to date, got: %v", err)
	}
}

func TestSelfUpdate_NetworkError(t *testing.T) {
	old := apiBaseURL
	apiBaseURL = "http://127.0.0.1:1" // nothing listening
	defer func() { apiBaseURL = old }()

	err := selfUpdate("v0.1.0", "/nonexistent/path/gtx")
	if err == nil {
		t.Error("expected error for network failure, got nil")
	}
}
