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

func TestIsForbidden(t *testing.T) {
	cases := []struct {
		name string
		r    rune
		want bool
	}{
		{"space is allowed", ' ', false},
		{"printable letter is allowed", 'a', false},
		{"control character is forbidden", '\x01', true},
		{"surrogate is forbidden", rune(0xD800), true},
		{"format character (soft hyphen) is allowed", rune(0x00AD), false},
		{"private use area is allowed", rune(0xE000), false},
		{"noncharacter FDD0 is forbidden", rune(0xFDD0), true},
		{"noncharacter FDEF is forbidden", rune(0xFDEF), true},
		{"noncharacter FFFE is forbidden", rune(0xFFFE), true},
		{"noncharacter FFFF is forbidden", rune(0xFFFF), true},
		{"noncharacter 1FFFE is forbidden", rune(0x1FFFE), true},
		{"unassigned code point is forbidden", rune(0x0378), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsForbidden(c.r); got != c.want {
				t.Fatalf("IsForbidden(%#x) = %v, want %v", c.r, got, c.want)
			}
		})
	}
}

func TestConfigurationUnit_String(t *testing.T) {
	cf := &ConfigurationUnit{
		Directives: []Directive{
			{Arguments: []string{"server_name", "example.com"}},
			{
				Arguments: []string{"database"},
				Subdirectives: []Directive{
					{Arguments: []string{"host", "localhost"}},
					{
						Arguments: []string{"pool"},
						Subdirectives: []Directive{
							{Arguments: []string{"max_size", "10"}},
						},
					},
				},
			},
		},
	}

	want := "<server_name> <example.com>\n" +
		"<database> [\n" +
		"    <host> <localhost>\n" +
		"    <pool> [\n" +
		"        <max_size> <10>\n" +
		"    ]\n" +
		"]\n"

	if got := cf.String(); got != want {
		t.Fatalf("String() =\n%q\nwant\n%q", got, want)
	}
}

func TestConfigurationUnit_String_Empty(t *testing.T) {
	cf := &ConfigurationUnit{}
	if got := cf.String(); got != "" {
		t.Fatalf("expected empty string for empty directives, got %q", got)
	}
}
