package confetti

// Options configures optional Confetti language extensions.
type Options struct {
	// CStyleComments enables Annex A: // and /* ... */ comment syntax.
	CStyleComments bool

	// ExpressionArguments enables Annex B: (expr) argument syntax with balanced parentheses.
	// the value of the argument is the content between the outermost parentheses.
	ExpressionArguments bool

	// PunctuatorArguments enables Annex C: self-delimiting punctuator argument tokens.
	// each string is a punctuator that will be recognized as a standalone argument.
	// longer punctuators are matched first (maximal munch); empty strings are ignored.
	// each punctuator must consist of argument characters only and must not contain
	// a reverse solidus or syntax reserved by another enabled extension; otherwise
	// parsing fails with a (non-*ParseError) error describing the bad punctuator.
	PunctuatorArguments []string
}
