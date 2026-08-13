package confetti

import "fmt"

// ParseError describes a syntax error and its position in the input.
// Line and Column are 1-based. Retrieve it with errors.As:
//
//	var perr *confetti.ParseError
//	if errors.As(err, &perr) {
//		fmt.Println(perr.Line, perr.Column, perr.Msg)
//	}
type ParseError struct {
	Line   int    // 1-based line of the offending input
	Column int    // 1-based column of the offending input
	Msg    string // description of the error, without position information
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("confetti: %s at line %d, column %d", e.Msg, e.Line, e.Column)
}
