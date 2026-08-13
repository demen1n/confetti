package confetti

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestParse_Basic(t *testing.T) {
	cfg, err := Parse("server {\n  host localhost\n  port 8080\n}\n")
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if len(cfg.Directives) != 1 {
		t.Fatalf("got %d top-level directives, want 1", len(cfg.Directives))
	}
	dir := cfg.Directives[0]
	if !reflect.DeepEqual(dir.Arguments, []string{"server"}) {
		t.Errorf("got arguments %v, want [server]", dir.Arguments)
	}
	if len(dir.Subdirectives) != 2 {
		t.Fatalf("got %d subdirectives, want 2", len(dir.Subdirectives))
	}
	if !reflect.DeepEqual(dir.Subdirectives[0].Arguments, []string{"host", "localhost"}) {
		t.Errorf("got subdirective %v, want [host localhost]", dir.Subdirectives[0].Arguments)
	}
}

func TestParseWithOptions_Extensions(t *testing.T) {
	opts := Options{
		CStyleComments:      true,
		PunctuatorArguments: []string{":=", "="},
	}
	cfg, err := ParseWithOptions("// comment\nuser:=smith\n", opts)
	if err != nil {
		t.Fatalf("ParseWithOptions error: %v", err)
	}
	if len(cfg.Directives) != 1 {
		t.Fatalf("got %d directives, want 1", len(cfg.Directives))
	}
	want := []string{"user", ":=", "smith"}
	if !reflect.DeepEqual(cfg.Directives[0].Arguments, want) {
		t.Errorf("got arguments %v, want %v", cfg.Directives[0].Arguments, want)
	}
}

func TestParse_ErrorIsParseError(t *testing.T) {
	tests := []struct {
		name     string
		src      string
		wantLine int
		wantCol  int
		wantMsg  string
	}{
		{
			name:     "lexer error position",
			src:      "ok directive\nbad \x01arg\n",
			wantLine: 2,
			wantCol:  5,
			wantMsg:  "forbidden character",
		},
		{
			name:     "parser error position",
			src:      "ok\n}\n",
			wantLine: 2,
			wantCol:  1,
			wantMsg:  "unexpected '}' without matching '{'",
		},
		{
			name:     "unterminated string",
			src:      "key \"unclosed\n",
			wantLine: 1,
			wantMsg:  "unexpected newline in single-quoted string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(tt.src)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			var perr *ParseError
			if !errors.As(err, &perr) {
				t.Fatalf("error %v (%T) is not a *ParseError", err, err)
			}
			if perr.Line != tt.wantLine {
				t.Errorf("got line %d, want %d", perr.Line, tt.wantLine)
			}
			if tt.wantCol != 0 && perr.Column != tt.wantCol {
				t.Errorf("got column %d, want %d", perr.Column, tt.wantCol)
			}
			if perr.Msg != tt.wantMsg {
				t.Errorf("got message %q, want %q", perr.Msg, tt.wantMsg)
			}
			if !strings.Contains(err.Error(), "confetti:") {
				t.Errorf("Error() = %q, want confetti: prefix", err.Error())
			}
		})
	}
}

func TestUnmarshalWithOptions(t *testing.T) {
	type Config struct {
		User string `conf:"user"`
	}
	opts := Options{CStyleComments: true}
	var got Config
	if err := UnmarshalWithOptions("/* who */ user smith\n", &got, opts); err != nil {
		t.Fatalf("UnmarshalWithOptions error: %v", err)
	}
	if got.User != "smith" {
		t.Errorf("got user %q, want smith", got.User)
	}
}

func TestTokenTypeString(t *testing.T) {
	tests := []struct {
		typ  TokenType
		want string
	}{
		{TokenEOF, "end of input"},
		{TokenNewline, "newline"},
		{TokenSemicolon, "';'"},
		{TokenLeftBrace, "'{'"},
		{TokenRightBrace, "'}'"},
		{TokenArgument, "argument"},
		{TokenComment, "comment"},
		{TokenLineContinuation, "line continuation"},
		{TokenType(99), "unknown token"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("TokenType(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}
