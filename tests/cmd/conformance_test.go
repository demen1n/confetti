package main

import (
	"os/exec"
	"testing"
)

func TestConformance(t *testing.T) {
	cmd := exec.Command("go", "run", "conformance.go", "-dir", defaultTestPath, "-v")
	output, err := cmd.CombinedOutput()
	t.Log(string(output))
	if err != nil {
		t.Fatalf("Conformance tests failed: %v", err)
	}
}
