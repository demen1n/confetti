package confetti

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Lexer tokenizes Confetti source text
type Lexer struct {
	input  string
	pos    int
	line   int
	column int
}

// NewLexer creates a new lexer
func NewLexer(input string) *Lexer {
	return &Lexer{
		input:  input,
		pos:    0,
		line:   1,
		column: 1,
	}
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
	// but if we haven't consumed any real tokens yet, check if it's truly at end
	if r == '\x1A' {
		// check if there's anything after Control-Z
		if l.pos+1 < len(l.input) {
			// control-Z in middle of file is forbidden
			return Token{}, fmt.Errorf("forbidden character at line %d, column %d", l.line, l.column)
		}
		// control-Z at actual end of file is treated as EOF
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

	// comment
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

	// update line counter
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
					// backslash at end of argument followed by newline is an error
					return Token{}, fmt.Errorf("illegal escape character")
				}
				// consume the line terminator
				term := l.advance()
				if term == '\r' && l.peek() == '\n' {
					l.advance()
				}
				l.line++
				l.column = 1

				// skip whitespace after line continuation
				l.skipWhitespace()

				// return special token for line continuation
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
				term := l.advance() // consume the line terminator
				// handle CRLF as single line terminator
				if term == '\r' && l.peek() == '\n' {
					l.advance() // consume the LF after CR
				}
				// after line continuation, update line counter
				l.line++
				l.column = 1
				continue // skip the newline, don't add to buffer
			}

			// escaped character
			if !IsWhitespace(next) && !IsLineTerminator(next) && !IsForbidden(next) {
				buf.WriteRune(next)
				l.advance()
				continue
			}

			return Token{}, fmt.Errorf("invalid escape in quoted string at line %d", l.line)
		}

		// now check for unescaped newlines (which are errors)
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
