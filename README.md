# Confetti for Go

A clean, fully conformant Go implementation of the [Confetti configuration language](https://confetti.hgs3.me/).

[![Go Reference](https://pkg.go.dev/badge/github.com/demen1n/confetti.svg)](https://pkg.go.dev/github.com/demen1n/confetti)
[![Go Report Card](https://goreportcard.com/badge/github.com/demen1n/confetti)](https://goreportcard.com/report/github.com/demen1n/confetti)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## What's Confetti?

Confetti is a minimal, human-friendly configuration format. Think of it as a simpler alternative to YAML or TOML — less magic, more clarity.

```confetti
# Simple directives
server_name example.com
port 8080

# Nested blocks
database {
    host localhost
    port 5432
    credentials {
        user admin
        password "secret123"
    }
}

# Quoted strings when you need them
message "Hello, World!"
```

## Features

- ✅ **100% conformant** with [Confetti 1.0.0 specification](https://confetti.hgs3.me/specification/)
- ✅ Passes all 194 official conformance tests (core + all Annex extensions)
- ✅ Full Unicode support (emojis, right-to-left text, etc.)
- ✅ All three optional Annex extensions implemented
- ✅ Clean, idiomatic Go API
- ✅ Zero dependencies

## Installation

```bash
go get github.com/demen1n/confetti
```

## Quick Start

```go
package main

import (
    "github.com/demen1n/confetti"
    "fmt"
    "log"
)

func main() {
    input := `
    server {
        host localhost
        port 8080
    }
    `

    parser, err := confetti.NewParser(input)
    if err != nil {
        log.Fatal(err)
    }

    config, err := parser.Parse()
    if err != nil {
        log.Fatal(err)
    }

    // Access the configuration
    for _, directive := range config.Directives {
        fmt.Printf("Directive: %v\n", directive.Arguments)
        for _, sub := range directive.Subdirectives {
            fmt.Printf("  Sub: %v\n", sub.Arguments)
        }
    }
}
```

## Decoding into Go structs

Use `Unmarshal` (or `Decode` with a pre-parsed `*ConfigurationUnit`) to populate typed Go structs — similar to `encoding/json`.

**Config file (`app.conf`):**
```confetti
host example.com
port 8080
debug true
tags foo bar baz

server web {
    timeout 30
}
server api {
    timeout 60
}
```

**Go code:**
```go
type Config struct {
    Host    string   `conf:"host"`
    Port    int      `conf:"port"`
    Debug   bool     `conf:"debug"`
    Tags    []string `conf:"tags"`
    Servers []Server `conf:"server"`
}

type Server struct {
    Name    string `conf:",arg"` // captures inline block argument ("web", "api")
    Timeout int    `conf:"timeout"`
}

data, _ := os.ReadFile("app.conf")
var cfg Config
if err := confetti.Unmarshal(string(data), &cfg); err != nil {
    log.Fatal(err)
}
// cfg.Host      → "example.com"
// cfg.Port      → 8080
// cfg.Debug     → true
// cfg.Tags      → ["foo", "bar", "baz"]
// cfg.Servers   → [{Name:"web", Timeout:30}, {Name:"api", Timeout:60}]
```

### Struct tags

| Tag | Meaning |
|-----|---------|
| `conf:"name"` | Map field to directive named `name` |
| `conf:",arg"` | Capture the inline args of a block directive |
| `conf:"-"` | Skip this field entirely |
| _(no tag)_ | Use the lowercase field name |

### Supported field types

| Go type | Source |
|---------|--------|
| `string` | First argument after the directive name |
| `int`, `int8` … `int64` | Parsed with `strconv.ParseInt` |
| `uint`, `uint8` … `uint64` | Parsed with `strconv.ParseUint` |
| `float32`, `float64` | Parsed with `strconv.ParseFloat` |
| `bool` | Parsed with `strconv.ParseBool` |
| `[]string` | All arguments after the directive name |
| `struct` / `*struct` | Decoded from the directive's subdirectives |
| `[]Struct` / `[]*Struct` | Each matching directive appends a new element |

---

## API

### Parsing

```go
// Parse from string (no extensions)
parser, err := confetti.NewParser(configString)
config, err := parser.Parse()

// Parse from file
data, err := os.ReadFile("config.conf")
parser, err := confetti.NewParser(string(data))
config, err := parser.Parse()

// Parse with extensions enabled
opts := confetti.Options{
    CStyleComments:      true,
    ExpressionArguments: true,
    PunctuatorArguments: []string{":=", "="},
}
parser, err := confetti.NewParserWithOptions(configString, opts)
config, err := parser.Parse()
```

### Data Structure

```go
type ConfigurationUnit struct {
    Directives []Directive
}

type Directive struct {
    Arguments     []string    // Directive arguments
    Subdirectives []Directive // Nested directives (if it's a block)
}
```

### Options

```go
type Options struct {
    // Annex A: enables // single-line and /* */ block comments.
    CStyleComments bool

    // Annex B: enables (expr) argument syntax with balanced parentheses.
    // The argument value is the content between the outermost parentheses.
    ExpressionArguments bool

    // Annex C: defines self-delimiting punctuator tokens.
    // Each string is recognized as a standalone argument (maximal munch —
    // longer punctuators take precedence regardless of order in the slice).
    PunctuatorArguments []string
}
```

## What's Supported

**Core language:**
- **Simple directives**: `key value1 value2`
- **Block directives**: `section { nested directives }`
- **Quoted strings**: `"hello world"` and `"""multi-line"""`
- **Escape sequences**: `\n`, `\t`, `\"`, etc.
- **Line continuations**: backslash at end of line continues on the next
- **Comments**: `# This is a comment`
- **Unicode**: Full support including emojis 👨‍🚀
- **Multiple terminators**: Both newline and `;` work

**Annex A — C-style comments** (opt-in via `Options.CStyleComments`):
- Single-line: `// comment`
- Block: `/* multi-line comment */`

**Annex B — Expression arguments** (opt-in via `Options.ExpressionArguments`):
- Parenthesised expressions with balanced nesting: `compute (1 + (2 * 3))`
- Value is the content without outer parentheses: `1 + (2 * 3)`

**Annex C — Punctuator arguments** (opt-in via `Options.PunctuatorArguments`):
- Self-delimiting tokens built from argument characters
- Example with `[":=", "="]`: `user:=smith` → three arguments: `user`, `:=`, `smith`
- Longer punctuators always win (maximal munch)

## Testing

Run the test suite:

```bash
go test -v
```

### Conformance Tests

Download and run the official Confetti conformance tests:

```bash
# Download latest tests from official repo
./download-tests.sh

# Run conformance tests
go run tests/cmd/conformance.go -dir ./tests/conformance -v
```

## License

MIT License - see [LICENSE](LICENSE) file for details.

This implementation is based on the [Confetti specification](https://confetti.hgs3.me/) by Henry G. Stratmann III.

## Contributing

Contributions welcome! Please:

1. Ensure all tests pass (`go test ./...`)
2. Run conformance tests (`go run tests/cmd/conformance.go`)
3. Follow existing code style
4. Add tests for new features

## Links

- [Confetti Specification](https://confetti.hgs3.me/specification/)
- [Official Confetti Repository](https://github.com/hgs3/confetti)
- [Conformance Test Suite](https://github.com/hgs3/confetti/tree/master/tests)
