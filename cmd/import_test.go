package cmd

import (
	"os"
	"testing"
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

func TestReadImportInputStdin(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
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
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
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
	old := os.Stdin
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = old })
	_ = w.Close()

	if _, err := readImportInput([]string{"-"}); err == nil {
		t.Fatal("expected error for empty stdin")
	}
}
