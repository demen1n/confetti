package confetti

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

// Lexer tokenizes Confetti source text
type Lexer struct {
	input        string
	pos          int
	line         int
	column       int
	opts         Options
	sortedPuncts []string // PunctuatorArguments sorted by length descending (maximal munch)
}

// NewLexer creates a new lexer with no extensions enabled.
func NewLexer(input string) *Lexer {
	return NewLexerWithOptions(input, Options{})
}

// NewLexerWithOptions creates a new lexer with the given extension options.
func NewLexerWithOptions(input string, opts Options) *Lexer {
	l := &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
		opts:   opts,
	}
	if len(opts.PunctuatorArguments) > 0 {
		l.sortedPuncts = make([]string, len(opts.PunctuatorArguments))
		copy(l.sortedPuncts, opts.PunctuatorArguments)
		sort.Slice(l.sortedPuncts, func(i, j int) bool {
			return len(l.sortedPuncts[i]) > len(l.sortedPuncts[j])
		})
	}
	return l
}

// NextToken returns the next token
func (l *Lexer) NextToken() (Token, error) {
	// check for malformed UTF-8 on first call
	if l.pos == 0 {
		if !ValidateUTF8(l.input) {
			return Token{}, fmt.Errorf("malformed UTF-8")
		}
		// skip BOM at the beginning of file
		if len(l.input) >= 3 && l.input[0:3] == "\xEF\xBB\xBF" {
			l.pos = 3
		}
	}

	l.skipWhitespace()

	if l.pos >= len(l.input) {
		return l.makeToken(TokenEOF, ""), nil
	}

	r := l.peek()

	// control-Z (SUB, 0x1A) after whitespace/at start of token is treated as EOF
	if r == '\x1A' {
		if l.pos+1 < len(l.input) {
			return Token{}, fmt.Errorf("forbidden character at line %d, column %d", l.line, l.column)
		}
		return l.makeToken(TokenEOF, ""), nil
	}

	// check for forbidden characters
	if IsForbidden(r) {
		return Token{}, fmt.Errorf("forbidden character at line %d, column %d", l.line, l.column)
	}

	// line terminator
	if IsLineTerminator(r) {
		return l.scanNewline()
	}

	// C-style comments (Annex A): // and /* */
	if l.opts.CStyleComments && r == '/' {
		next := l.peekSecond()
		if next == '/' {
			return l.scanCStyleLineComment()
		}
		if next == '*' {
			return l.scanCStyleBlockComment()
		}
	}

	// hash comment
	if r == '#' {
		return l.scanComment()
	}

	// semicolon
	if r == ';' {
		tok := l.makeToken(TokenSemicolon, ";")
		l.advance()
		return tok, nil
	}

	// left brace
	if r == '{' {
		tok := l.makeToken(TokenLeftBrace, "{")
		l.advance()
		return tok, nil
	}

	// right brace
	if r == '}' {
		tok := l.makeToken(TokenRightBrace, "}")
		l.advance()
		return tok, nil
	}

	// expression argument (Annex B)
	if l.opts.ExpressionArguments && r == '(' {
		return l.scanExpressionArgument()
	}

	// punctuator argument (Annex C) — checked before simple argument
	if len(l.sortedPuncts) > 0 && l.matchesPunctuator() {
		return l.scanPunctuatorArgument()
	}

	// quoted argument
	if r == '"' {
		return l.scanQuotedArgument()
	}

	// simple argument
	if IsArgumentChar(r) {
		return l.scanSimpleArgument()
	}

	return Token{}, fmt.Errorf("unexpected character '%c' at line %d, column %d", r, l.line, l.column)
}

func (l *Lexer) peek() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos:])
	return r
}

// peekSecond returns the rune after the current one without advancing.
func (l *Lexer) peekSecond() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	_, size := utf8.DecodeRuneInString(l.input[l.pos:])
	if l.pos+size >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.pos+size:])
	return r
}

func (l *Lexer) advance() rune {
	if l.pos >= len(l.input) {
		return 0
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	l.column++
	return r
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		r := l.peek()
		if !IsWhitespace(r) {
			break
		}
		l.advance()
	}
}

func (l *Lexer) makeToken(typ TokenType, value string) Token {
	return Token{
		Type:   typ,
		Value:  value,
		Line:   l.line,
		Column: l.column,
	}
}

func (l *Lexer) scanNewline() (Token, error) {
	tok := l.makeToken(TokenNewline, "")
	r := l.advance()

	// handle CRLF as single newline
	if r == '\r' && l.peek() == '\n' {
		l.advance()
	}

	l.line++
	l.column = 1

	return tok, nil
}

func (l *Lexer) scanComment() (Token, error) {
	start := l.pos
	l.advance() // skip '#'

	for l.pos < len(l.input) {
		r := l.peek()
		if IsLineTerminator(r) {
			break
		}
		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in comment at line %d", l.line)
		}
		l.advance()
	}

	value := l.input[start:l.pos]
	return l.makeToken(TokenComment, value), nil
}

// scanCStyleLineComment scans a // single-line comment (Annex A).
func (l *Lexer) scanCStyleLineComment() (Token, error) {
	start := l.pos
	l.advance() // skip first '/'
	l.advance() // skip second '/'

	for l.pos < len(l.input) {
		r := l.peek()
		if IsLineTerminator(r) {
			break
		}
		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in comment at line %d", l.line)
		}
		l.advance()
	}

	value := l.input[start:l.pos]
	return l.makeToken(TokenComment, value), nil
}

// scanCStyleBlockComment scans a /* ... */ block comment (Annex A).
func (l *Lexer) scanCStyleBlockComment() (Token, error) {
	startLine := l.line
	start := l.pos
	l.advance() // skip '/'
	l.advance() // skip '*'

	for l.pos < len(l.input) {
		r := l.peek()

		// check for closing */
		if r == '*' && l.peekSecond() == '/' {
			l.advance() // skip '*'
			l.advance() // skip '/'
			value := l.input[start:l.pos]
			return l.makeToken(TokenComment, value), nil
		}

		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in comment at line %d", l.line)
		}

		if IsLineTerminator(r) {
			consumed := l.advance()
			if consumed == '\r' && l.peek() == '\n' {
				l.advance()
			}
			l.line++
			l.column = 1
			continue
		}

		l.advance()
	}

	return Token{}, fmt.Errorf("unterminated block comment starting at line %d", startLine)
}

// scanExpressionArgument scans a (expr) argument with balanced parentheses (Annex B).
func (l *Lexer) scanExpressionArgument() (Token, error) {
	tok := l.makeToken(TokenArgument, "")
	l.advance() // skip opening '('

	var buf strings.Builder
	depth := 1

	for l.pos < len(l.input) {
		r := l.peek()

		if r == '(' {
			depth++
			buf.WriteRune(r)
			l.advance()
			continue
		}

		if r == ')' {
			depth--
			if depth == 0 {
				l.advance() // skip closing ')'
				tok.Value = buf.String()
				return tok, nil
			}
			buf.WriteRune(r)
			l.advance()
			continue
		}

		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in expression at line %d", l.line)
		}

		if IsLineTerminator(r) {
			consumed := l.advance()
			if consumed == '\r' && l.peek() == '\n' {
				l.advance()
			}
			l.line++
			l.column = 1
			buf.WriteRune('\n')
			continue
		}

		buf.WriteRune(r)
		l.advance()
	}

	return Token{}, fmt.Errorf("unterminated expression argument at line %d", l.line)
}

// matchesPunctuator reports whether the current position starts a punctuator argument.
func (l *Lexer) matchesPunctuator() bool {
	for _, p := range l.sortedPuncts {
		if strings.HasPrefix(l.input[l.pos:], p) {
			return true
		}
	}
	return false
}

// scanPunctuatorArgument emits the longest matching punctuator argument (Annex C).
func (l *Lexer) scanPunctuatorArgument() (Token, error) {
	tok := l.makeToken(TokenArgument, "")
	for _, punct := range l.sortedPuncts {
		if strings.HasPrefix(l.input[l.pos:], punct) {
			tok.Value = punct
			for range punct { // iterates once per rune
				l.advance()
			}
			return tok, nil
		}
	}
	return Token{}, fmt.Errorf("internal error: no punctuator matched at line %d", l.line)
}

func (l *Lexer) scanSimpleArgument() (Token, error) {
	var buf strings.Builder
	tok := l.makeToken(TokenArgument, "")

	for l.pos < len(l.input) {
		r := l.peek()

		// escape sequence
		if r == '\\' {
			l.advance() // skip '\'
			next := l.peek()

			// line continuation - only allowed if buffer is empty (standalone backslash)
			if IsLineTerminator(next) {
				if buf.Len() > 0 {
					return Token{}, fmt.Errorf("illegal escape character")
				}
				term := l.advance()
				if term == '\r' && l.peek() == '\n' {
					l.advance()
				}
				l.line++
				l.column = 1
				l.skipWhitespace()
				return l.makeToken(TokenLineContinuation, ""), nil
			}

			// escaped character
			if !IsWhitespace(next) && !IsLineTerminator(next) && !IsForbidden(next) {
				buf.WriteRune(next)
				l.advance()
				continue
			}

			return Token{}, fmt.Errorf("invalid escape sequence at line %d, column %d", l.line, l.column)
		}

		// C-style comment start terminates the argument (Annex A)
		if l.opts.CStyleComments && r == '/' && l.peekSecond() == '/' {
			break
		}
		if l.opts.CStyleComments && r == '/' && l.peekSecond() == '*' {
			break
		}

		// expression argument start terminates the simple argument (Annex B)
		if l.opts.ExpressionArguments && r == '(' {
			break
		}

		// punctuator argument start terminates the simple argument (Annex C)
		if len(l.sortedPuncts) > 0 && l.matchesPunctuator() {
			break
		}

		if !IsArgumentChar(r) {
			break
		}

		buf.WriteRune(r)
		l.advance()
	}

	tok.Value = buf.String()
	return tok, nil
}

func (l *Lexer) scanQuotedArgument() (Token, error) {
	tok := l.makeToken(TokenArgument, "")
	l.advance() // skip opening '"'

	// check for triple-quoted
	if l.peek() == '"' {
		l.advance()
		if l.peek() == '"' {
			l.advance()
			return l.scanTripleQuoted()
		}
		// empty single-quoted string
		tok.Value = ""
		return tok, nil
	}

	return l.scanSingleQuoted()
}

func (l *Lexer) scanSingleQuoted() (Token, error) {
	var buf strings.Builder
	tok := l.makeToken(TokenArgument, "")

	for l.pos < len(l.input) {
		r := l.peek()

		if r == '"' {
			l.advance() // skip closing '"'
			tok.Value = buf.String()
			return tok, nil
		}

		// escape sequence - must check BEFORE checking for newline
		if r == '\\' {
			l.advance() // skip the backslash
			next := l.peek()

			// line continuation: backslash followed by line terminator
			if IsLineTerminator(next) {
				term := l.advance()
				if term == '\r' && l.peek() == '\n' {
					l.advance()
				}
				l.line++
				l.column = 1
				continue
			}

			// escaped character
			if !IsWhitespace(next) && !IsLineTerminator(next) && !IsForbidden(next) {
				buf.WriteRune(next)
				l.advance()
				continue
			}

			return Token{}, fmt.Errorf("invalid escape in quoted string at line %d", l.line)
		}

		// unescaped newlines are errors in single-quoted strings
		if IsLineTerminator(r) {
			return Token{}, fmt.Errorf("unexpected newline in single-quoted string at line %d", l.line)
		}

		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in string at line %d", l.line)
		}

		buf.WriteRune(r)
		l.advance()
	}

	return Token{}, fmt.Errorf("unterminated quoted string at line %d", l.line)
}

func (l *Lexer) scanTripleQuoted() (Token, error) {
	var buf strings.Builder
	tok := l.makeToken(TokenArgument, "")

	for l.pos < len(l.input) {
		r := l.peek()

		// check for closing """
		if r == '"' && l.pos+2 < len(l.input) {
			next1, _ := utf8.DecodeRuneInString(l.input[l.pos+1:])
			next2, _ := utf8.DecodeRuneInString(l.input[l.pos+2:])
			if next1 == '"' && next2 == '"' {
				l.advance()
				l.advance()
				l.advance()
				tok.Value = buf.String()
				return tok, nil
			}
		}

		// escape sequence
		if r == '\\' {
			l.advance()
			next := l.peek()
			if !IsWhitespace(next) && !IsLineTerminator(next) && !IsForbidden(next) {
				buf.WriteRune(next)
				l.advance()
				continue
			}
			return Token{}, fmt.Errorf("invalid escape in triple-quoted string at line %d", l.line)
		}

		if IsForbidden(r) {
			return Token{}, fmt.Errorf("forbidden character in string at line %d", l.line)
		}

		buf.WriteRune(r)
		l.advance()
	}

	return Token{}, fmt.Errorf("unterminated triple-quoted string at line %d", l.line)
}
