# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run unit tests
go test -v ./...

# Run a single test
go test -v -run TestName .

# Run conformance tests
go run tests/cmd/conformance.go -dir ./tests/conformance -v

# Download/refresh official conformance tests
./download-tests.sh

# Build
go build ./...
```

## Architecture

This is a **Confetti configuration language parser** implemented as a Go library. It follows a classic compiler pipeline:

```
Input string → Lexer → Tokens → Parser → ConfigurationUnit (AST)
```

### Core Files

- **`types.go`** — Token types (`TokenArgument`, `TokenNewline`, `TokenLeftBrace`, etc.), AST types (`ConfigurationUnit`, `Directive`), and Unicode predicate helpers (`IsLineTerminator`, `IsForbidden`, `IsArgumentChar`, etc.)
- **`lexer.go`** — Single-pass `Lexer` struct; `NextToken()` is the primary method. Handles BOM stripping, CRLF normalization, escape sequences, quoted strings (single and triple), and line continuations.
- **`parser.go`** — Recursive descent `Parser` struct; `Parse()` returns `*ConfigurationUnit`. Uses one-token lookahead via `current Token`. Calls `parseDirectives()` → `parseDirective()` → `parseArguments()` + `parseBlock()`.

### Public API

```go
parser, err := confetti.NewParser(input)   // validates UTF-8, creates Lexer+Parser
config, err := parser.Parse()             // returns *ConfigurationUnit
// config.Directives []Directive
// directive.Arguments []string
// directive.Subdirectives []Directive
```

### Test Structure

- **`*_test.go`** — Unit tests for lexer and parser
- **`tests/cmd/conformance.go`** — Conformance test runner (standalone `main` package, not part of the library)
- **`tests/conformance/`** — Official spec test fixtures (gitignored; download with `./download-tests.sh`)
  - `.conf` = input, `.pass`/`.fail` = expected outcome
  - `.ext_*` files mark tests for unimplemented extensions (C-style comments, expression arguments, punctuator arguments) — these are skipped

### Extensions (Annex A/B/C)

All three optional spec extensions are implemented and enabled via `Options`:

```go
opts := confetti.Options{
    CStyleComments:      true,              // Annex A: // and /* */ comments
    ExpressionArguments: true,              // Annex B: (expr) arguments with balanced parens
    PunctuatorArguments: []string{":=","="}, // Annex C: self-delimiting punctuators (maximal munch)
}
parser, err := confetti.NewParserWithOptions(input, opts)
```

- `NewParser` keeps zero-extension behaviour (backward compatible)
- Punctuators are sorted by length internally (longest wins); order in the slice doesn't matter
- Conformance test runner reads punctuators from each `.ext_punctuator_arguments` marker file (one per line)

### Key Invariants

- Zero external dependencies — standard library only (`sort` added for punctuator ordering)
- All Unicode line terminators (LF, CR, VT, FF, NEL, LS, PS) are recognized
- Forbidden characters (control chars, surrogates, noncharacters) are rejected at the lexer level with line/column error messages
- Extensions are off by default; `NewParser` remains unchanged for existing users