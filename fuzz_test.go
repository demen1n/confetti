package confetti

import (
	"errors"
	"testing"
	"unicode/utf8"
)

var fuzzOptions = []Options{
	{},
	{
		CStyleComments:      true,
		ExpressionArguments: true,
		PunctuatorArguments: []string{":=", "=", "→"},
	},
}

// FuzzParse checks that parsing never panics or hangs, that syntax errors are
// always *ParseError with a valid position, and that a successful parse only
// produces directives with at least one argument.
func FuzzParse(f *testing.F) {
	seeds := []string{
		"",
		"key value\n",
		"server web {\n  listen 80; tls on\n}\n",
		"a { b { c } }",
		"x \"quoted \\\" arg\" \"\"\"triple\nquoted\"\"\"\n",
		"probe eth0 \\\n  eth1\n",
		"# comment\nfoo\x1a",
		"\uFEFFbom\r\ncrlf\r\n",
		"x // line\n/* block\n */ y\n",
		"compute (1 + (2 * 3)) ())\n",
		"user:=smith a=b c→d\n",
		"unterminated \"",
		"lonely }",
		"bad \x01 char",
		"\xff",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, input string) {
		for _, opts := range fuzzOptions {
			cfg, err := ParseWithOptions(input, opts)
			if err != nil {
				var perr *ParseError
				if !errors.As(err, &perr) {
					t.Fatalf("error is not *ParseError: %T %v", err, err)
				}
				if perr.Line < 1 || perr.Column < 1 {
					t.Fatalf("invalid error position %d:%d: %v", perr.Line, perr.Column, err)
				}
				continue
			}
			if !utf8.ValidString(input) {
				t.Fatalf("malformed UTF-8 input was accepted")
			}
			checkDirectives(t, cfg.Directives)
			_ = cfg.String()
		}
	})
}

func checkDirectives(t *testing.T, dirs []Directive) {
	t.Helper()
	for _, d := range dirs {
		if len(d.Arguments) == 0 {
			t.Fatalf("directive without arguments: %+v", d)
		}
		checkDirectives(t, d.Subdirectives)
	}
}
