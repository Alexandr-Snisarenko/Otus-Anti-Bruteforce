package version

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrintVersion_WithLdflags(t *testing.T) {
	t.Parallel()

	// Prepare a small main program inside the module (required to import internal package)
	tmpDir, err := os.MkdirTemp(".", "ver_test_*")
	if err != nil {
		t.Fatalf("create tmp dir: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(tmpDir) })
	mainPath := filepath.Join(tmpDir, "main.go")
	mainSrc := `package main

import "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/version"

func main() {
    version.PrintVersion()
}
`
	if err := os.WriteFile(mainPath, []byte(mainSrc), 0o644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}

	// Values to inject via -ldflags
	wantRel := "vX.Y.Z-test"
	wantDate := "2025-12-19T00:00:00Z"
	wantHash := "deadbeef"

	pkg := "github.com/Alexandr-Snisarenko/Otus-Anti-Bruteforce/internal/version"
	ld := fmt.Sprintf("-X %s.Release=%s -X %s.BuildDate=%s -X %s.GitHash=%s",
		pkg, wantRel, pkg, wantDate, pkg, wantHash)

	// Run `go run -ldflags '<ld>' main.go` and capture stdout
	cmd := exec.Command("go", "run", "-ldflags", ld, mainPath)
	// ensure execution inside repo so internal package import is allowed
	cmd.Dir = "."
	cmd.Env = os.Environ()
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("command failed: %v, stderr=%s", err, string(ee.Stderr))
		}
		t.Fatalf("command failed: %v", err)
	}

	// Parse JSON output
	var got struct {
		Release   string `json:"Release"`
		BuildDate string `json:"BuildDate"`
		GitHash   string `json:"GitHash"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("json unmarshal: %v, out=%s", err, string(out))
	}

	if got.Release != wantRel || got.BuildDate != wantDate || got.GitHash != wantHash {
		t.Fatalf("unexpected version: got=%+v want=%+v", got, struct{ R, D, H string }{wantRel, wantDate, wantHash})
	}
}
