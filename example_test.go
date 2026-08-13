package confetti_test

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/demen1n/confetti"
)

func ExampleParse() {
	config, err := confetti.Parse("server {\n  host localhost\n  port 8080\n}")
	if err != nil {
		log.Fatal(err)
	}
	for _, d := range config.Directives {
		fmt.Println(d.Arguments)
		for _, sub := range d.Subdirectives {
			fmt.Println(" ", sub.Arguments)
		}
	}
	// Output:
	// [server]
	//   [host localhost]
	//   [port 8080]
}

func ExampleParseWithOptions() {
	opts := confetti.Options{
		CStyleComments:      true,
		PunctuatorArguments: []string{":=", "="},
	}
	config, err := confetti.ParseWithOptions("// login\nuser:=smith", opts)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(config.Directives[0].Arguments)
	// Output: [user := smith]
}

func ExampleUnmarshal() {
	type Config struct {
		Host    string        `conf:"host"`
		Port    int           `conf:"port"`
		Timeout time.Duration `conf:"timeout"`
		Tags    []string      `conf:"tags"`
	}

	src := `
host example.com
port 8080
timeout 1m30s
tags web api
`
	var cfg Config
	if err := confetti.Unmarshal(src, &cfg); err != nil {
		log.Fatal(err)
	}
	fmt.Println(cfg.Host, cfg.Port, cfg.Timeout, cfg.Tags)
	// Output: example.com 8080 1m30s [web api]
}

func ExampleParseError() {
	_, err := confetti.Parse(`key "unterminated`)

	var perr *confetti.ParseError
	if errors.As(err, &perr) {
		fmt.Printf("%d:%d %s\n", perr.Line, perr.Column, perr.Msg)
	}
	// Output: 1:18 unterminated quoted string
}
