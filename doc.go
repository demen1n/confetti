// Package confetti implements a parser for the Confetti configuration
// language (https://confetti.hgs3.me/).
//
// Confetti is a minimal, human-friendly configuration format. A document
// is a sequence of directives: whitespace-separated arguments optionally
// followed by a block of subdirectives in curly braces:
//
//	server_name example.com
//
//	database {
//	    host localhost
//	    port 5432
//	}
//
// # Parsing
//
// [Parse] returns the document as a tree of [Directive] values:
//
//	config, err := confetti.Parse(input)
//	if err != nil {
//		log.Fatal(err)
//	}
//	for _, d := range config.Directives {
//		fmt.Println(d.Arguments, len(d.Subdirectives))
//	}
//
// # Decoding into structs
//
// [Unmarshal] populates a struct from a document, similar to encoding/json.
// Fields are matched by the "conf" struct tag, or by the lowercased field
// name when the tag is absent:
//
//	type Config struct {
//		Host    string        `conf:"host"`
//		Port    int           `conf:"port"`
//		Timeout time.Duration `conf:"timeout"`
//		Tags    []string      `conf:"tags"`
//	}
//	var cfg Config
//	err := confetti.Unmarshal(input, &cfg)
//
// The tag `conf:",arg"` captures the inline arguments of a block directive,
// and `conf:"-"` skips a field. See [Decode] for decoding an already-parsed
// [ConfigurationUnit].
//
// # Extensions
//
// The three optional extensions from the specification's annexes — C-style
// comments, expression arguments, and punctuator arguments — are disabled
// by default and enabled per call via [Options] with [ParseWithOptions] or
// [UnmarshalWithOptions].
//
// # Errors
//
// Syntax errors are reported as [*ParseError] carrying the 1-based line and
// column of the offending input; retrieve it with errors.As.
package confetti
