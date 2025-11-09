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
- ✅ Passes all 165 official conformance tests
- ✅ Full Unicode support (emojis, right-to-left text, etc.)
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

## API

### Parsing

```go
// Parse from string
parser, err := confetti.NewParser(configString)
config, err := parser.Parse()

// Parse from file
data, err := os.ReadFile("config.conf")
parser, err := confetti.NewParser(string(data))
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

## What's Supported

- **Simple directives**: `key value1 value2`
- **Block directives**: `section { nested directives }`
- **Quoted strings**: `"hello world"` and `"""multi-line"""`
- **Escape sequences**: `\n`, `\t`, `\"`, etc.
- **Line continuations**: `foo \ ` continues on next line
- **Comments**: `# This is a comment`
- **Unicode**: Full support including emojis 👨‍🚀
- **Multiple terminators**: Both newline and `;` work

## What's Not (Yet) Supported

The following optional extensions from the spec are not implemented:

- C-style comments (`/* */` and `//`)
- Expression arguments `(1 + 2)`
- Punctuator arguments (`:=`, `+=`, etc.)

These are Annex features that may be added in the future. The core language is fully supported.

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
