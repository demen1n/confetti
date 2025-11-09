package confetti

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// ConfigurationUnit represents the entire Confetti configuration
type ConfigurationUnit struct {
	Directives []Directive
}

func (cf *ConfigurationUnit) String() string {
	var sb strings.Builder
	printDirectives(&sb, cf.Directives, 0)
	return sb.String()
}

func printDirectives(sb *strings.Builder, directives []Directive, indent int) {
	for _, dir := range directives {
		// print indentation
		for i := 0; i < indent; i++ {
			sb.WriteString("    ")
		}

		// print arguments in angle brackets
		for i, arg := range dir.Arguments {
			sb.WriteString("<")
			sb.WriteString(arg)
			sb.WriteString(">")
			// add space only between arguments, not after the last one
			if i < len(dir.Arguments)-1 {
				sb.WriteString(" ")
			}
		}

		// print subdirectives
		if len(dir.Subdirectives) > 0 {
			sb.WriteString(" [\n")
			printDirectives(sb, dir.Subdirectives, indent+1)
			for i := 0; i < indent; i++ {
				sb.WriteString("    ")
			}
			sb.WriteString("]")
		}

		sb.WriteString("\n")
	}
}

// Directive represents a single directive with arguments and optional subdirectives
type Directive struct {
	Arguments     []string
	Subdirectives []Directive
}

// TokenType represents the type of token
type TokenType int

const (
	TokenEOF TokenType = iota
	TokenNewline
	TokenSemicolon
	TokenLeftBrace
	TokenRightBrace
	TokenArgument
	TokenComment
	TokenLineContinuation // special token for standalone backslash before newline
)

// Token represents a lexical token
type Token struct {
	Type   TokenType
	Value  string
	Line   int
	Column int
}

// ValidateUTF8 checks if the input string is valid UTF-8
func ValidateUTF8(s string) bool {
	return utf8.ValidString(s)
}

// IsWhitespace checks if rune is whitespace but not a line terminator
func IsWhitespace(r rune) bool {
	if IsLineTerminator(r) {
		return false
	}
	return unicode.Is(unicode.White_Space, r)
}

// IsLineTerminator checks if rune is a line terminator
func IsLineTerminator(r rune) bool {
	switch r {
	case '\u000A', // LF
		'\u000B', // VT
		'\u000C', // FF
		'\u000D', // CR
		'\u0085', // NEL
		'\u2028', // LS
		'\u2029': // PS
		return true
	}
	return false
}

// IsForbidden checks if rune is a forbidden character
func IsForbidden(r rune) bool {
	// whitespace is never forbidden
	if unicode.Is(unicode.White_Space, r) {
		return false
	}

	// format characters (category Cf) are allowed - they're for invisible formatting
	if unicode.Is(unicode.Cf, r) {
		return false
	}

	// control characters (category Cc) are forbidden
	if unicode.Is(unicode.Cc, r) {
		return true
	}

	// surrogate characters
	if unicode.Is(unicode.Cs, r) {
		return true
	}

	// private Use Area characters are allowed (implementation choice)
	if unicode.In(r, unicode.Co) {
		return false // explicitly allow
	}

	// noncharacter code points: U+FDD0..U+FDEF and U+nFFFE, U+nFFFF for n=0..16
	if r >= 0xFDD0 && r <= 0xFDEF {
		return true
	}
	if (r & 0xFFFE) == 0xFFFE { // ends with FFFE or FFFF
		return true
	}

	// check if character is in a defined Unicode category
	// if not in any category, it's unassigned (truly unassigned, not private use)
	if !unicode.IsPrint(r) && !unicode.IsControl(r) && !unicode.IsSpace(r) && !unicode.In(r, unicode.Co) {
		return true
	}

	return false
}

// IsReservedPunctuator checks if rune is a reserved punctuator
func IsReservedPunctuator(r rune) bool {
	switch r {
	case '"', '#', ';', '{', '}':
		return true
	}
	return false
}

// IsArgumentChar checks if rune can be part of an unquoted argument
func IsArgumentChar(r rune) bool {
	if IsWhitespace(r) || IsLineTerminator(r) || IsReservedPunctuator(r) || IsForbidden(r) {
		return false
	}
	return true
}
