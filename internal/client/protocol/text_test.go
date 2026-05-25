package protocol

import "testing"

func TestStripCardURLLines(t *testing.T) {
	input := "Sample assistant response.\nhttp://googleusercontent.com/card_content/0\nkept line\nhttp://googleusercontent.com/video_gen_chip/1\n"
	want := "Sample assistant response.\nkept line"
	if got := StripCardURLLines(input); got != want {
		t.Fatalf("StripCardURLLines = %q, want %q", got, want)
	}
}

func TestStripCardURLLines_PreservesInlineURL(t *testing.T) {
	input := "See http://googleusercontent.com/card_content/0 for details"
	if got := StripCardURLLines(input); got != input {
		t.Fatalf("StripCardURLLines = %q, want %q", got, input)
	}
}
