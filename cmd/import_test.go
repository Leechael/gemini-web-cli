package cmd

import (
	"io"
	"os"
	"strings"
	"testing"
	"time"
)

func TestReadImportInputArgument(t *testing.T) {
	got, err := readImportInput([]string{"__Secure-1PSID=abc; SID=x"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "__Secure-1PSID=abc; SID=x" {
		t.Fatalf("got %q", got)
	}
}

func withImportStdin(t *testing.T, src importStdinSource) {
	t.Helper()
	old := importStdin
	importStdin = src
	t.Cleanup(func() { importStdin = old })
}

func TestReadImportInputStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	withImportStdin(t, r)
	if _, err := w.WriteString("  __Secure-1PSID=abc; SID=x \n"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	got, err := readImportInput(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "__Secure-1PSID=abc; SID=x" {
		t.Fatalf("got %q", got)
	}
}

func TestReadImportInputDash(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	withImportStdin(t, r)
	if _, err := w.WriteString("__Secure-1PSID=fromdash"); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()

	got, err := readImportInput([]string{"-"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "__Secure-1PSID=fromdash" {
		t.Fatalf("got %q", got)
	}
}

func TestReadImportInputEmptyStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	withImportStdin(t, r)
	_ = w.Close()

	if _, err := readImportInput([]string{"-"}); err == nil {
		t.Fatal("expected error for empty stdin")
	}
}

func TestReadImportInputTTYRequiresArg(t *testing.T) {
	withImportStdin(t, stubStdin{Reader: strings.NewReader("should-not-read"), mode: os.ModeCharDevice})
	_, err := readImportInput(nil)
	if err == nil || !strings.Contains(err.Error(), "cookie string required") {
		t.Fatalf("err = %v, want cookie string required", err)
	}
}

type stubStdin struct {
	io.Reader
	mode os.FileMode
}

func (s stubStdin) Stat() (os.FileInfo, error) {
	return stubFileInfo{mode: s.mode}, nil
}

type stubFileInfo struct{ mode os.FileMode }

func (stubFileInfo) Name() string        { return "stdin" }
func (stubFileInfo) Size() int64         { return 0 }
func (i stubFileInfo) Mode() os.FileMode { return i.mode }
func (stubFileInfo) ModTime() time.Time  { return time.Time{} }
func (stubFileInfo) IsDir() bool         { return false }
func (stubFileInfo) Sys() any            { return nil }
