package confetti

// Parse parses a Confetti document with no extensions enabled.
//
// Syntax errors are reported as *ParseError with line and column information.
func Parse(input string) (*ConfigurationUnit, error) {
	return ParseWithOptions(input, Options{})
}

// ParseWithOptions parses a Confetti document with the given extension options.
//
// Syntax errors are reported as *ParseError with line and column information.
func ParseWithOptions(input string, opts Options) (*ConfigurationUnit, error) {
	p, err := newParser(input, opts)
	if err != nil {
		return nil, err
	}
	return p.Parse()
}
