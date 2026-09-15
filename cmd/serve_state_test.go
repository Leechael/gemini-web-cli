package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveCookiesJSONWithStateDirPriority(t *testing.T) {
	oldCookiesJSON := cookiesJSON
	t.Cleanup(func() { cookiesJSON = oldCookiesJSON })
	cookiesJSON = nil
	t.Setenv(envCookiesPath, filepath.Join(t.TempDir(), "env-cookies.json"))

	stateDir := t.TempDir()
	stateCookies := filepath.Join(stateDir, "cookies.json")
	if err := os.WriteFile(stateCookies, []byte(`{"cookies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	path, source := resolveCookiesJSONWithStateDir(stateDir)
	if path != stateCookies || source != "state-dir" {
		t.Fatalf("path/source = %q/%q, want %q/state-dir", path, source, stateCookies)
	}

	explicit := filepath.Join(t.TempDir(), "explicit.json")
	cookiesJSON = []string{explicit}
	path, source = resolveCookiesJSONWithStateDir(stateDir)
	if path != explicit || source != "--cookies-json" {
		t.Fatalf("explicit path/source = %q/%q", path, source)
	}
}

func TestResolveCookiePathsExpandsDirectories(t *testing.T) {
	oldCookiesJSON := cookiesJSON
	t.Cleanup(func() { cookiesJSON = oldCookiesJSON })

	accountsDir := t.TempDir()
	for _, name := range []string{"b.json", "a.json", "not-cookies.txt"} {
		if err := os.WriteFile(filepath.Join(accountsDir, name), []byte(`{"cookies":{}}`), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	single := filepath.Join(t.TempDir(), "single.json")
	if err := os.WriteFile(single, []byte(`{"cookies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Flag entries: directory expands to sorted *.json, files pass through.
	cookiesJSON = []string{accountsDir, single}
	paths, source := resolveCookiePathsWithStateDir("")
	want := []string{
		filepath.Join(accountsDir, "a.json"),
		filepath.Join(accountsDir, "b.json"),
		single,
	}
	if source != "--cookies-json" {
		t.Fatalf("source = %q", source)
	}
	if len(paths) != len(want) {
		t.Fatalf("paths = %v, want %v", paths, want)
	}
	for i := range want {
		if paths[i] != want[i] {
			t.Fatalf("paths[%d] = %q, want %q", i, paths[i], want[i])
		}
	}

	// Env var accepts a path list.
	cookiesJSON = nil
	t.Setenv(envCookiesPath, single+string(os.PathListSeparator)+accountsDir)
	paths, source = resolveCookiePathsWithStateDir("")
	if source != "$"+envCookiesPath {
		t.Fatalf("source = %q", source)
	}
	if len(paths) != 3 || paths[0] != single {
		t.Fatalf("env paths = %v", paths)
	}

	// Docker default: GEMINI_WEB_COOKIES_JSON_PATH=/cookies may be a file
	// (no .json suffix) or a directory of *.json files.
	fileAtCookies := filepath.Join(t.TempDir(), "cookies")
	if err := os.WriteFile(fileAtCookies, []byte(`{"cookies":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(envCookiesPath, fileAtCookies)
	paths, source = resolveCookiePathsWithStateDir("")
	if source != "$"+envCookiesPath || len(paths) != 1 || paths[0] != fileAtCookies {
		t.Fatalf("file-at-cookies paths/source = %v/%q", paths, source)
	}

	t.Setenv(envCookiesPath, accountsDir)
	paths, source = resolveCookiePathsWithStateDir("")
	if source != "$"+envCookiesPath || len(paths) != 2 || paths[0] != filepath.Join(accountsDir, "a.json") {
		t.Fatalf("dir-at-cookies paths/source = %v/%q", paths, source)
	}
}

func TestClientConfigsErrorsOnEmptyExplicitDir(t *testing.T) {
	oldCookiesJSON := cookiesJSON
	t.Cleanup(func() { cookiesJSON = oldCookiesJSON })

	cookiesJSON = []string{t.TempDir()} // no *.json inside
	if _, _, err := clientConfigsWithStateDir(""); err == nil {
		t.Fatal("expected error for an explicitly configured directory with no cookie files")
	}
}

func TestDefaultCookiesPathWithPathListEnv(t *testing.T) {
	first := filepath.Join(t.TempDir(), "first.json")
	t.Setenv(envCookiesPath, first+string(os.PathListSeparator)+"second.json")
	if got := defaultCookiesPath(); got != first {
		t.Fatalf("defaultCookiesPath = %q, want %q", got, first)
	}

	dir := t.TempDir()
	t.Setenv(envCookiesPath, dir)
	if got, want := defaultCookiesPath(), filepath.Join(dir, "cookies.json"); got != want {
		t.Fatalf("defaultCookiesPath = %q, want %q", got, want)
	}
}

func TestClientConfigFromFlagsErrorsOnEmptyExplicitDir(t *testing.T) {
	oldCookiesJSON := cookiesJSON
	t.Cleanup(func() { cookiesJSON = oldCookiesJSON })

	cookiesJSON = []string{t.TempDir()} // no *.json inside
	if _, _, _, err := clientConfigFromFlagsWithStateDir(""); err == nil {
		t.Fatal("expected error for an explicitly configured directory with no cookie files")
	}
}

func TestServeBindHostPort(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		t.Setenv(envServeHost, "")
		t.Setenv(envServePort, "")
		if got := serveBindHost("127.0.0.1", false); got != "127.0.0.1" {
			t.Fatalf("host = %q", got)
		}
		port, err := serveBindPort(8080, false)
		if err != nil || port != 8080 {
			t.Fatalf("port = %d err=%v", port, err)
		}
	})
	t.Run("env", func(t *testing.T) {
		t.Setenv(envServeHost, "0.0.0.0")
		t.Setenv(envServePort, "9000")
		if got := serveBindHost("127.0.0.1", false); got != "0.0.0.0" {
			t.Fatalf("host = %q", got)
		}
		port, err := serveBindPort(8080, false)
		if err != nil || port != 9000 {
			t.Fatalf("port = %d err=%v", port, err)
		}
	})
	t.Run("flag wins", func(t *testing.T) {
		t.Setenv(envServeHost, "0.0.0.0")
		t.Setenv(envServePort, "9000")
		if got := serveBindHost("127.0.0.1", true); got != "127.0.0.1" {
			t.Fatalf("host = %q", got)
		}
		port, err := serveBindPort(8080, true)
		if err != nil || port != 8080 {
			t.Fatalf("port = %d err=%v", port, err)
		}
	})
	t.Run("invalid port env", func(t *testing.T) {
		t.Setenv(envServePort, "nope")
		if _, err := serveBindPort(8080, false); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestServeBindStateDir(t *testing.T) {
	t.Setenv(envServeStateDir, "")
	if got := serveBindStateDir("", false); got != "" {
		t.Fatalf("default state-dir = %q", got)
	}

	t.Setenv(envServeStateDir, "/state")
	if got := serveBindStateDir("", false); got != "/state" {
		t.Fatalf("env state-dir = %q", got)
	}
	if got := serveBindStateDir("/explicit", true); got != "/explicit" {
		t.Fatalf("flag state-dir = %q", got)
	}
}
