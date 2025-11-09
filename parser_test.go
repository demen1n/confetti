package confetti

import (
	"reflect"
	"testing"
)

func parseOK(t *testing.T, src string) *ConfigurationUnit {
	t.Helper()
	p, err := NewParser(src)
	if err != nil {
		t.Fatalf("parser init error: %v", err)
	}
	unit, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return unit
}

func TestParser_SimpleDirective_Semicolon(t *testing.T) {
	src := "listen 80;"
	u := parseOK(t, src)

	want := &ConfigurationUnit{
		Directives: []Directive{
			{Arguments: []string{"listen", "80"}},
		},
	}
	if !reflect.DeepEqual(u, want) {
		t.Fatalf("AST mismatch:\n got: %#v\nwant: %#v", u, want)
	}
}

func TestParser_SimpleDirective_Newline(t *testing.T) {
	src := "root /var/www\n"
	u := parseOK(t, src)

	want := &ConfigurationUnit{
		Directives: []Directive{
			{Arguments: []string{"root", "/var/www"}},
		},
	}
	if !reflect.DeepEqual(u, want) {
		t.Fatalf("AST mismatch:\n got: %#v\nwant: %#v", u, want)
	}
}

func TestParser_BlockDirective(t *testing.T) {
	src := `
server {
    listen 80;
    server_name example.com
}
`
	u := parseOK(t, src)

	want := &ConfigurationUnit{
		Directives: []Directive{
			{
				Arguments: []string{"server"},
				Subdirectives: []Directive{
					{Arguments: []string{"listen", "80"}},
					{Arguments: []string{"server_name", "example.com"}},
				},
			},
		},
	}

	if !reflect.DeepEqual(u, want) {
		t.Fatalf("AST mismatch:\n got: %#v\nwant: %#v", u, want)
	}
}

func TestParser_NestedBlocks(t *testing.T) {
	src := `
http {
  server {
    location "/" {
      proxy_pass "http://127.0.0.1:9000";
    }
  }
}
`
	u := parseOK(t, src)

	// spot check the shape
	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 top-level directive, got %d", len(u.Directives))
	}
	http := u.Directives[0]
	if http.Arguments[0] != "http" {
		t.Fatalf("expected 'http' block")
	}
	if len(http.Subdirectives) != 1 || http.Subdirectives[0].Arguments[0] != "server" {
		t.Fatalf("expected 'server' inside 'http'")
	}
	loc := http.Subdirectives[0].Subdirectives[0]
	if loc.Arguments[0] != "location" || loc.Arguments[1] != "/" {
		t.Fatalf("expected location \"/\" block")
	}
	if got := loc.Subdirectives[0].Arguments; !reflect.DeepEqual(got, []string{"proxy_pass", "http://127.0.0.1:9000"}) {
		t.Fatalf("unexpected inner directive args: %v", got)
	}
}

func TestParser_LastDirectiveWithoutTerminatorAtEOF(t *testing.T) {
	src := `key value`
	u := parseOK(t, src)

	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(u.Directives))
	}
	if args := u.Directives[0].Arguments; !reflect.DeepEqual(args, []string{"key", "value"}) {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestParser_CommentsAreSkipped(t *testing.T) {
	src := `
# top comment
key value; # inline
# another
`
	u := parseOK(t, src)

	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(u.Directives))
	}
	if args := u.Directives[0].Arguments; !reflect.DeepEqual(args, []string{"key", "value"}) {
		t.Fatalf("unexpected args: %v", args)
	}
}

func TestParser_Error_NoArguments(t *testing.T) {
	// lone semicolon/newline shouldn't be a directive
	src := ";\n{\n}\n"
	p, err := NewParser(src)
	if err != nil {
		t.Fatalf("init parser: %v", err)
	}
	_, err = p.Parse()
	if err == nil {
		t.Fatalf("expected error for directive without arguments")
	}
}

func TestParser_ParenthesisAsArgument(t *testing.T) {
	src := "key value )"
	p, err := NewParser(src)
	if err != nil {
		t.Fatalf("init parser: %v", err)
	}
	u, err := p.Parse()
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	want := []string{"key", "value", ")"}
	if !reflect.DeepEqual(u.Directives[0].Arguments, want) {
		t.Fatalf("expected %v, got %v", want, u.Directives[0].Arguments)
	}
}

func TestParser_MultipleDirectivesOnOneLine(t *testing.T) {
	src := "key1 val1; key2 val2; key3 val3"
	u := parseOK(t, src)

	if len(u.Directives) != 3 {
		t.Fatalf("expected 3 directives, got %d", len(u.Directives))
	}

	expected := [][]string{
		{"key1", "val1"},
		{"key2", "val2"},
		{"key3", "val3"},
	}

	for i, exp := range expected {
		if !reflect.DeepEqual(u.Directives[i].Arguments, exp) {
			t.Errorf("directive %d: expected %v, got %v", i, exp, u.Directives[i].Arguments)
		}
	}
}

func TestParser_BlockWithOptionalSemicolon(t *testing.T) {
	src := `server {
    listen 80
};`
	u := parseOK(t, src)

	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(u.Directives))
	}
}

func TestParser_EmptyBlock(t *testing.T) {
	src := "empty {}"
	u := parseOK(t, src)

	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(u.Directives))
	}
	if len(u.Directives[0].Subdirectives) != 0 {
		t.Fatalf("expected empty subdirectives, got %d", len(u.Directives[0].Subdirectives))
	}
}

func TestParser_BlockWithNewlinesBeforeBrace(t *testing.T) {
	src := `server

{
    listen 80
}`

	// debug: посмотрим на токены
	lx := NewLexer(src)
	for i := 0; i < 10; i++ {
		tok, err := lx.NextToken()
		if err != nil {
			t.Fatalf("lexer error: %v", err)
		}
		// t.Logf("Token %d: type=%v, value=%q", i, tok.Type, tok.Value)
		if tok.Type == TokenEOF {
			break
		}
	}

	u := parseOK(t, src)

	if len(u.Directives) != 1 {
		t.Fatalf("expected 1 directive, got %d", len(u.Directives))
	}
	if u.Directives[0].Arguments[0] != "server" {
		t.Fatalf("expected 'server' directive")
	}
}

func TestParser_DeeplyNestedBlocks(t *testing.T) {
	src := `a { b { c { d { e { f value } } } } }`
	u := parseOK(t, src)

	// navigate down the tree
	curr := u.Directives[0]
	depth := 0
	for len(curr.Subdirectives) > 0 {
		curr = curr.Subdirectives[0]
		depth++
	}

	if depth != 5 {
		t.Fatalf("expected depth 5, got %d", depth)
	}
	if !reflect.DeepEqual(curr.Arguments, []string{"f", "value"}) {
		t.Fatalf("unexpected leaf directive: %v", curr.Arguments)
	}
}

func TestParser_MultipleBlocksAtSameLevel(t *testing.T) {
	src := `
server {
    listen 80
}
server {
    listen 443
}
`
	u := parseOK(t, src)

	if len(u.Directives) != 2 {
		t.Fatalf("expected 2 directives, got %d", len(u.Directives))
	}

	if u.Directives[0].Subdirectives[0].Arguments[1] != "80" {
		t.Errorf("first server port wrong")
	}
	if u.Directives[1].Subdirectives[0].Arguments[1] != "443" {
		t.Errorf("second server port wrong")
	}
}

func TestParser_ComplexRealWorldExample(t *testing.T) {
	src := `
# Configuration file
global {
    timeout 30
    retry 3
}

server "web-01" {
    host 192.168.1.10
    port 8080

    location "/api" {
        proxy_pass "http://backend:9000"
        timeout 60
    }

    location "/static" {
        root "/var/www/static"
    }
}

server "web-02" {
    host 192.168.1.11
    port 8080
}
`
	u := parseOK(t, src)

	if len(u.Directives) != 3 {
		t.Fatalf("expected 3 top-level directives, got %d", len(u.Directives))
	}

	// check global block
	if u.Directives[0].Arguments[0] != "global" {
		t.Errorf("expected 'global' block")
	}
	if len(u.Directives[0].Subdirectives) != 2 {
		t.Errorf("global should have 2 subdirectives")
	}

	// check first server
	server1 := u.Directives[1]
	if server1.Arguments[0] != "server" || server1.Arguments[1] != "web-01" {
		t.Errorf("expected server web-01")
	}
	if len(server1.Subdirectives) != 4 { // host, port, location, location
		t.Errorf("server should have 4 subdirectives, got %d", len(server1.Subdirectives))
	}
}

func TestParser_ArgumentsWithSpecialChars(t *testing.T) {
	// test that various characters are valid in arguments
	src := `path /usr/local/bin
email user@example.com
version v1.2.3-beta+build.123
url https://example.com:8080/path?query=value&foo=bar
`
	u := parseOK(t, src)

	if len(u.Directives) != 4 {
		t.Fatalf("expected 4 directives, got %d", len(u.Directives))
	}

	expectations := []string{
		"/usr/local/bin",
		"user@example.com",
		"v1.2.3-beta+build.123",
		"https://example.com:8080/path?query=value&foo=bar",
	}

	for i, exp := range expectations {
		if u.Directives[i].Arguments[1] != exp {
			t.Errorf("directive %d: expected %q, got %q", i, exp, u.Directives[i].Arguments[1])
		}
	}
}

func TestParser_EmptyLines(t *testing.T) {
	src := `

key1 value1


key2 value2



key3 value3

`
	u := parseOK(t, src)

	if len(u.Directives) != 3 {
		t.Fatalf("expected 3 directives, got %d", len(u.Directives))
	}
}

func TestParser_OnlyComments(t *testing.T) {
	src := `# Just comments
# Nothing else
# Should parse fine`
	u := parseOK(t, src)

	if len(u.Directives) != 0 {
		t.Fatalf("expected 0 directives, got %d", len(u.Directives))
	}
}

// not sure for this
// func TestParser_Error_UnmatchedClosingBrace(t *testing.T) {
// 	src := "key value }"
// 	p, err := NewParser(src)
// 	if err != nil {
// 		t.Fatalf("init parser: %v", err)
// 	}
// 	_, err = p.Parse()
// 	// this should actually succeed - } is a valid argument character!
// 	if err != nil {
// 		t.Logf("Got error (may be valid): %v", err)
// 	}
// }

func TestParser_Error_UnmatchedOpeningBrace(t *testing.T) {
	src := "server { listen 80"
	p, err := NewParser(src)
	if err != nil {
		t.Fatalf("init parser: %v", err)
	}
	_, err = p.Parse()
	if err == nil {
		t.Fatalf("expected error for unmatched opening brace")
	}
}

func TestParser_Error_EmptyInput(t *testing.T) {
	src := ""
	u := parseOK(t, src)
	if len(u.Directives) != 0 {
		t.Fatalf("expected 0 directives for empty input, got %d", len(u.Directives))
	}
}

func TestParser_Error_OnlyWhitespace(t *testing.T) {
	src := "   \t\n\n  \t  \n"
	u := parseOK(t, src)
	if len(u.Directives) != 0 {
		t.Fatalf("expected 0 directives for whitespace-only input, got %d", len(u.Directives))
	}
}
