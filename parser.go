package confetti

import "fmt"

// maxNestingDepth caps block nesting to keep the recursive-descent parser
// from overflowing the goroutine stack on adversarial input.
const maxNestingDepth = 5000

// Parser parses Confetti tokens into a ConfigurationUnit
type Parser struct {
	lexer   *Lexer
	current Token
	depth   int
}

// NewParser creates a new parser with no extensions enabled.
//
// Deprecated: Use Parse instead.
func NewParser(input string) (*Parser, error) {
	return newParser(input, Options{})
}

// NewParserWithOptions creates a new parser with the given extension options.
//
// Deprecated: Use ParseWithOptions instead.
func NewParserWithOptions(input string, opts Options) (*Parser, error) {
	return newParser(input, opts)
}

func newParser(input string, opts Options) (*Parser, error) {
	p := &Parser{
		lexer: NewLexerWithOptions(input, opts),
	}

	// load first token
	if err := p.advance(); err != nil {
		return nil, err
	}

	return p, nil
}

// errf returns a *ParseError at the current token's position.
func (p *Parser) errf(format string, args ...any) error {
	return &ParseError{Line: p.current.Line, Column: p.current.Column, Msg: fmt.Sprintf(format, args...)}
}

// Parse parses the input and returns a ConfigurationUnit
func (p *Parser) Parse() (*ConfigurationUnit, error) {
	directives, err := p.parseDirectives(false) // false = top-level, no closing brace expected
	if err != nil {
		return nil, err
	}

	return &ConfigurationUnit{
		Directives: directives,
	}, nil
}

func (p *Parser) advance() error {
	for {
		tok, err := p.lexer.NextToken()
		if err != nil {
			return err
		}

		// skip comments
		if tok.Type == TokenComment {
			continue
		}

		p.current = tok
		break
	}

	return nil
}

func (p *Parser) parseDirectives(insideBlock bool) ([]Directive, error) {
	var directives []Directive

	for {
		// skip empty lines
		if p.current.Type == TokenNewline {
			if err := p.advance(); err != nil {
				return nil, err
			}
			continue
		}

		if p.current.Type == TokenEOF {
			break
		}

		// closing brace
		if p.current.Type == TokenRightBrace {
			if insideBlock {
				break // expected closing brace
			}
			return nil, p.errf("unexpected '}' without matching '{'")
		}

		directive, err := p.parseDirective()
		if err != nil {
			return nil, err
		}

		directives = append(directives, directive)
	}

	return directives, nil
}

func (p *Parser) parseDirective() (Directive, error) {
	args, err := p.parseArguments()
	if err != nil {
		return Directive{}, err
	}

	if len(args) == 0 {
		return Directive{}, p.errf("directive must have at least one argument")
	}

	directive := Directive{
		Arguments: args,
	}

	// check what comes after arguments (possibly with newlines before block)
	// save current position to check for block
	hasNewlines := p.current.Type == TokenNewline

	// skip newlines to peek ahead
	for p.current.Type == TokenNewline {
		if err := p.advance(); err != nil {
			return Directive{}, err
		}
	}

	// block directive case: { follows (possibly after newlines)
	if p.current.Type == TokenLeftBrace {
		subdirs, err := p.parseBlock()
		if err != nil {
			return Directive{}, err
		}
		directive.Subdirectives = subdirs

		// optional semicolon after block
		if p.current.Type == TokenSemicolon {
			if err := p.advance(); err != nil {
				return Directive{}, err
			}
		}

		return directive, nil
	}

	// simple directive case: we already consumed newlines above (if any)
	// just check we're at valid terminator
	if hasNewlines {
		// already consumed newlines, directive is done
		return directive, nil
	}

	// must have semicolon then
	if p.current.Type == TokenSemicolon {
		if err := p.advance(); err != nil {
			return Directive{}, err
		}
		return directive, nil
	}

	// if we're at } or EOF, directive is complete
	if p.current.Type == TokenRightBrace || p.current.Type == TokenEOF {
		return directive, nil
	}

	return Directive{}, p.errf("expected newline, semicolon, or block after directive, got %s", p.current.Type)
}

func (p *Parser) parseArguments() ([]string, error) {
	var args []string

	for p.current.Type == TokenArgument || p.current.Type == TokenLineContinuation {
		// skip line continuation tokens
		if p.current.Type == TokenLineContinuation {
			if err := p.advance(); err != nil {
				return nil, err
			}
			continue
		}

		args = append(args, p.current.Value)
		if err := p.advance(); err != nil {
			return nil, err
		}
	}

	return args, nil
}

func (p *Parser) parseBlock() ([]Directive, error) {
	// consume '{'
	if p.current.Type != TokenLeftBrace {
		return nil, p.errf("expected '{', got %s", p.current.Type)
	}

	p.depth++
	defer func() { p.depth-- }()
	if p.depth > maxNestingDepth {
		return nil, p.errf("exceeded maximum nesting depth of %d", maxNestingDepth)
	}

	if err := p.advance(); err != nil {
		return nil, err
	}

	// parse subdirectives
	subdirs, err := p.parseDirectives(true) // true = inside block
	if err != nil {
		return nil, err
	}

	// consume '}'
	if p.current.Type != TokenRightBrace {
		return nil, p.errf("expected '}', got %s", p.current.Type)
	}

	if err := p.advance(); err != nil {
		return nil, err
	}

	return subdirs, nil
}
