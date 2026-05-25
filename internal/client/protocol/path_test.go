package protocol

import "testing"

func TestArrayAt(t *testing.T) {
	root := []any{[]any{"zero", []any{"target"}}}
	got, ok := ArrayAt(root, 0, 1)
	if !ok {
		t.Fatalf("ArrayAt ok = false")
	}
	if got[0] != "target" {
		t.Fatalf("ArrayAt = %v", got)
	}
}

func TestArrayAt_MissingPath(t *testing.T) {
	root := []any{[]any{"zero"}}
	if _, ok := ArrayAt(root, 0, 2); ok {
		t.Fatalf("ArrayAt missing path ok = true")
	}
	if _, ok := ArrayAt(root, 0, 0); ok {
		t.Fatalf("ArrayAt string value ok = true")
	}
}

func TestValueAt(t *testing.T) {
	root := []any{[]any{"value"}}
	got, ok := ValueAt(root, 0, 0)
	if !ok {
		t.Fatalf("ValueAt ok = false")
	}
	if got != "value" {
		t.Fatalf("ValueAt = %v, want value", got)
	}
}

func TestStringAt(t *testing.T) {
	root := []any{[]any{"value", 1}}
	if got := StringAt(root, 0, 0); got != "value" {
		t.Fatalf("StringAt = %q, want value", got)
	}
	if got := StringAt(root, 0, 1); got != "" {
		t.Fatalf("StringAt non-string = %q, want empty", got)
	}
}

func TestIntAt(t *testing.T) {
	root := []any{[]any{float64(42), 1.5, "nope"}}
	if got := IntAt(root, 0, 0); got != 42 {
		t.Fatalf("IntAt = %d, want 42", got)
	}
	if got := IntAt(root, 0, 1); got != 0 {
		t.Fatalf("IntAt non-integral = %d, want 0", got)
	}
	if got := IntAt(root, 0, 2); got != 0 {
		t.Fatalf("IntAt non-number = %d, want 0", got)
	}
}

func TestBoolAt(t *testing.T) {
	root := []any{[]any{true, "nope"}}
	if got := BoolAt(root, 0, 0); !got {
		t.Fatalf("BoolAt = false, want true")
	}
	if got := BoolAt(root, 0, 1); got {
		t.Fatalf("BoolAt non-bool = true, want false")
	}
}

func TestFirstString(t *testing.T) {
	if got := FirstString("", "fallback", "later"); got != "fallback" {
		t.Fatalf("FirstString = %q, want fallback", got)
	}
	if got := FirstString("", ""); got != "" {
		t.Fatalf("FirstString empty = %q, want empty", got)
	}
}
