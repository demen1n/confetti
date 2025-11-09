package confetti

import "testing"

func TestIsArgumentChar_Basic(t *testing.T) {
	for _, r := range []rune{'a', 'Z', '0', '_', '-', '.', ':'} {
		if !IsArgumentChar(r) {
			t.Fatalf("expected %q to be argument char", r)
		}
	}
	for _, r := range []rune{' ', '\t', '\n', '"', '#', '{', '}', ';'} {
		if IsArgumentChar(r) {
			t.Fatalf("expected %q to NOT be argument char", r)
		}
	}
}

func TestIsLineTerminator(t *testing.T) {
	for _, r := range []rune{'\n', '\r'} {
		if !IsLineTerminator(r) {
			t.Fatalf("expected %q to be line terminator", r)
		}
	}
	if IsLineTerminator('a') {
		t.Fatalf("letter must not be a line terminator")
	}
}
