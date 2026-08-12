package main

import (
	"os/exec"
	"testing"
)

func TestConformance(t *testing.T) {
	// defaultTestPath is relative to the repo root, but this test (and the "go run"
	// child process it spawns) runs with the package directory as its working
	// directory, so it must be adjusted accordingly.
	cmd := exec.Command("go", "run", "conformance.go", "-dir", "../conformance", "-v")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("Conformance tests failed: %v", err)
	}
}
