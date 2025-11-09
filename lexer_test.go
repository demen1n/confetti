package confetti

import (
	"reflect"
	"testing"
)

func collectTokens(t *testing.T, src string) ([]Token, error) {
	t.Helper()
	lx := NewLexer(src)

	var toks []Token
	for {
		tok, err := lx.NextToken()
		if err != nil {
			return nil, err
		}
		toks = append(toks, tok)
		if tok.Type == TokenEOF {
			break
		}
	}
	return toks, nil
}

func TestLexer_SimpleTokens(t *testing.T) {
	src := `server listen 80; # comment
{ key "value" }`

	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	var got []TokenType
	for _, tk := range toks {
		got = append(got, tk.Type)
	}

	want := []TokenType{
		TokenArgument, TokenArgument, TokenArgument, TokenSemicolon, TokenComment, TokenNewline,
		TokenLeftBrace, TokenArgument, TokenArgument, TokenRightBrace, TokenEOF,
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("token types mismatch:\n got: %v\nwant: %v", got, want)
	}
}

func TestLexer_CommentToken(t *testing.T) {
	src := `# first
value # mid
# last`

	lx := NewLexer(src)

	// 1st: comment
	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tok.Type != TokenComment {
		t.Fatalf("expected TokenComment, got %v", tok.Type)
	}

	// 2nd: newline
	tok, _ = lx.NextToken()
	if tok.Type != TokenNewline {
		t.Fatalf("expected TokenNewline, got %v", tok.Type)
	}

	// 3rd: "value"
	tok, _ = lx.NextToken()
	if tok.Type != TokenArgument || tok.Value != "value" {
		t.Fatalf("expected argument 'value', got %v %q", tok.Type, tok.Value)
	}

	// 4th: space then comment token
	tok, _ = lx.NextToken()
	if tok.Type != TokenComment {
		t.Fatalf("expected TokenComment, got %v", tok.Type)
	}
}

func TestLexer_QuotedArguments_SingleAndTriple(t *testing.T) {
	src := "\"hello world\"\n\"\"\"multi\nline\ntext\"\"\""

	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	if toks[0].Type != TokenArgument || toks[0].Value != "hello world" {
		t.Fatalf("single-quoted mismatch, got %v %q", toks[0].Type, toks[0].Value)
	}
	if toks[1].Type != TokenNewline {
		t.Fatalf("expected newline after first string")
	}
	if toks[2].Type != TokenArgument || toks[2].Value != "multi\nline\ntext" {
		t.Fatalf("triple-quoted mismatch, got %v %q", toks[2].Type, toks[2].Value)
	}
}

func TestLexer_UnexpectedChar(t *testing.T) {
	// include a reserved punctuator inside a bare word -> should stop before it
	src := "key{"
	lx := NewLexer(src)

	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("unexpected error for first token: %v", err)
	}
	if tok.Type != TokenArgument || tok.Value != "key" {
		t.Fatalf("expected 'key' argument, got %v %q", tok.Type, tok.Value)
	}

	tok, err = lx.NextToken()
	if err != nil {
		t.Fatalf("unexpected error for second token: %v", err)
	}
	if tok.Type != TokenLeftBrace {
		t.Fatalf("expected left brace, got %v", tok.Type)
	}
}

func TestLexer_UnterminatedSingleQuoted(t *testing.T) {
	src := "\"oops\n"
	lx := NewLexer(src)

	_, err := lx.NextToken()
	if err == nil {
		t.Fatalf("expected error for unterminated quoted string")
	}
}

func TestLexer_UnterminatedTripleQuoted(t *testing.T) {
	src := "\"\"\"never ending..."
	lx := NewLexer(src)

	_, err := lx.NextToken()
	if err == nil {
		t.Fatalf("expected error for unterminated triple-quoted string")
	}
}

func TestLexer_CRLF_Handling(t *testing.T) {
	src := "key value\r\nfoo bar\r\n"
	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	// should have: key, value, \n, foo, bar, \n, EOF = 7 tokens
	if len(toks) != 7 {
		t.Fatalf("expected 7 tokens, got %d", len(toks))
	}

	// check that CRLF is treated as single newline
	if toks[2].Type != TokenNewline || toks[5].Type != TokenNewline {
		t.Fatalf("CRLF not properly converted to single newline")
	}
}

func TestLexer_EmptyQuotedString(t *testing.T) {
	src := `key ""`
	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	if toks[1].Type != TokenArgument || toks[1].Value != "" {
		t.Fatalf("expected empty string argument, got %v %q", toks[1].Type, toks[1].Value)
	}
}

func TestLexer_EscapeInSingleQuoted(t *testing.T) {
	src := `"Hello \"World\""`
	lx := NewLexer(src)
	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	expected := `Hello "World"`
	if tok.Value != expected {
		t.Fatalf("expected %q, got %q", expected, tok.Value)
	}
}

func TestLexer_LineContinuationInQuoted(t *testing.T) {
	src := "\"Hello \\\nWorld\""
	lx := NewLexer(src)
	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	expected := "Hello World"
	if tok.Value != expected {
		t.Fatalf("expected %q, got %q", expected, tok.Value)
	}
}

func TestLexer_TripleQuotedWithQuotes(t *testing.T) {
	src := `"""He said "Hello"!"""`
	lx := NewLexer(src)
	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	expected := `He said "Hello"!`
	if tok.Value != expected {
		t.Fatalf("expected %q, got %q", expected, tok.Value)
	}
}

func TestLexer_MultipleEscapesInRow(t *testing.T) {
	src := `\\n\\t`
	lx := NewLexer(src)
	tok, err := lx.NextToken()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	// \\ -> \, n stays n, \\ -> \, t stays t
	expected := `\n\t`
	if tok.Value != expected {
		t.Fatalf("expected %q, got %q", expected, tok.Value)
	}
}

func TestLexer_UnicodeLineTerminators(t *testing.T) {
	// test various Unicode line terminators
	terminators := []string{
		"\u000A", // LF
		"\u000B", // VT
		"\u000C", // FF
		"\u000D", // CR
		"\u0085", // NEL
		"\u2028", // LS
		"\u2029", // PS
	}

	for _, term := range terminators {
		src := "key" + term + "value"
		toks, err := collectTokens(t, src)
		if err != nil {
			t.Fatalf("lexer error for terminator %q: %v", term, err)
		}

		// should have: key, newline, value, EOF
		if len(toks) != 4 {
			t.Fatalf("expected 4 tokens for %q, got %d", term, len(toks))
		}
		if toks[1].Type != TokenNewline {
			t.Fatalf("terminator %q not recognized as newline", term)
		}
	}
}

func TestLexer_CommentAtEOF(t *testing.T) {
	src := "key value # comment without newline"
	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	// key, value, comment, EOF
	if len(toks) != 4 {
		t.Fatalf("expected 4 tokens, got %d", len(toks))
	}
	if toks[2].Type != TokenComment {
		t.Fatalf("expected comment token")
	}
}

func TestLexer_MixedWhitespace(t *testing.T) {
	src := "key\t  \t  value"
	toks, err := collectTokens(t, src)
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	if toks[0].Value != "key" || toks[1].Value != "value" {
		t.Fatalf("whitespace not properly skipped")
	}
}
