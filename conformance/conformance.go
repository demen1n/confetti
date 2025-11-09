package main

import (
	"confetti"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	testDir := flag.String("dir", "./tests/conformance", "directory with conformance tests")
	verbose := flag.Bool("v", false, "verbose output")
	flag.Parse()

	// find all .conf files
	confFiles, err := filepath.Glob(filepath.Join(*testDir, "*.conf"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error globbing files: %v\n", err)
		os.Exit(1)
	}

	if len(confFiles) == 0 {
		fmt.Fprintf(os.Stderr, "No .conf files found in %s\n", *testDir)
		os.Exit(1)
	}

	passed := 0
	failed := 0
	skipped := 0

	for _, confFile := range confFiles {
		baseName := strings.TrimSuffix(confFile, ".conf")
		testName := filepath.Base(baseName)

		// check for extension markers
		hasExtC := fileExists(baseName + ".ext_c_style_comments")
		hasExtExpr := fileExists(baseName + ".ext_expression_arguments")
		hasExtPunct := fileExists(baseName + ".ext_punctuator_arguments")

		if hasExtC || hasExtExpr || hasExtPunct {
			if *verbose {
				fmt.Printf("SKIP %s (requires extensions)\n", testName)
			}
			skipped++
			continue
		}

		// read input
		input, err := os.ReadFile(confFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", confFile, err)
			failed++
			continue
		}

		// parse
		parser, err := confetti.NewParser(string(input))
		if err != nil {
			// lexer error
			if fileExists(baseName + ".fail") {
				if *verbose {
					fmt.Printf("PASS %s (failed as expected: %v)\n", testName, err)
				}
				passed++
			} else {
				fmt.Printf("FAIL %s: unexpected lexer error: %v\n", testName, err)
				failed++
			}
			continue
		}

		unit, err := parser.Parse()
		if err != nil {
			// parser error
			if fileExists(baseName + ".fail") {
				if *verbose {
					fmt.Printf("PASS %s (failed as expected: %v)\n", testName, err)
				}
				passed++
			} else {
				fmt.Printf("FAIL %s: unexpected parser error: %v\n", testName, err)
				failed++
			}
			continue
		}

		// success - check if .pass file exists
		passFile := baseName + ".pass"
		if !fileExists(passFile) {
			fmt.Printf("FAIL %s: parsed successfully but no .pass file found\n", testName)
			failed++
			continue
		}

		// compare output
		expected, err := os.ReadFile(passFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", passFile, err)
			failed++
			continue
		}

		actual := unit.String()
		expectedStr := strings.TrimSpace(string(expected))
		actualStr := strings.TrimSpace(actual)

		if expectedStr != actualStr {
			fmt.Printf("FAIL %s: output mismatch\n", testName)
			if *verbose {
				fmt.Println("Expected:")
				fmt.Println(expectedStr)
				fmt.Println("\nActual:")
				fmt.Println(actualStr)
				fmt.Println("\n=== Character-by-character comparison ===")
				showDiff(expectedStr, actualStr)
			}
			failed++
		} else {
			if *verbose {
				fmt.Printf("PASS %s\n", testName)
			}
			passed++
		}
	}

	// summary
	fmt.Printf("\n===== Results =====\n")
	fmt.Printf("Passed:  %d\n", passed)
	fmt.Printf("Failed:  %d\n", failed)
	fmt.Printf("Skipped: %d (extensions not supported)\n", skipped)
	fmt.Printf("Total:   %d\n", passed+failed+skipped)

	if failed > 0 {
		os.Exit(1)
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func showDiff(expected, actual string) {
	expLines := strings.Split(expected, "\n")
	actLines := strings.Split(actual, "\n")

	maxLines := len(expLines)
	if len(actLines) > maxLines {
		maxLines = len(actLines)
	}

	for i := 0; i < maxLines; i++ {
		var expLine, actLine string
		if i < len(expLines) {
			expLine = expLines[i]
		}
		if i < len(actLines) {
			actLine = actLines[i]
		}

		if expLine != actLine {
			fmt.Printf("Line %d differs:\n", i+1)
			fmt.Printf("  Expected: %q (len=%d)\n", expLine, len(expLine))
			fmt.Printf("  Actual:   %q (len=%d)\n", actLine, len(actLine))

			// Show hex for first difference
			minLen := len(expLine)
			if len(actLine) < minLen {
				minLen = len(actLine)
			}
			for j := 0; j < minLen; j++ {
				if expLine[j] != actLine[j] {
					fmt.Printf("  First diff at char %d: expected %q (0x%02x), got %q (0x%02x)\n",
						j, expLine[j], expLine[j], actLine[j], actLine[j])
					break
				}
			}
			if len(expLine) != len(actLine) {
				fmt.Printf("  Length differs: expected %d, got %d\n", len(expLine), len(actLine))
			}
		}
	}

	if len(expLines) != len(actLines) {
		fmt.Printf("\nTotal lines: expected %d, got %d\n", len(expLines), len(actLines))
	}
}
