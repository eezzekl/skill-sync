package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// canonicalModulePath is the module path that external consumers must be able to
// resolve, and it has to match the repository URL exactly.
const canonicalModulePath = "github.com/eezzekl/skill-sync"

// TestGoModDeclaresCanonicalModulePath guards an invariant that neither the
// compiler nor the rest of the suite can observe. A wrong module path is
// internally consistent — every import agrees with go.mod — so everything builds
// and passes while `go install` fails for every consumer resolving the module
// through the proxy. Only an assertion against the repository URL catches it.
func TestGoModDeclaresCanonicalModulePath(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "go.mod"))
	if err != nil {
		t.Fatalf("failed to read go.mod: %v", err)
	}

	var declared string
	for _, line := range strings.Split(string(data), "\n") {
		if rest, ok := strings.CutPrefix(strings.TrimSpace(line), "module "); ok {
			declared = strings.TrimSpace(rest)
			break
		}
	}

	if declared == "" {
		t.Fatal("go.mod declares no module path")
	}

	if declared != canonicalModulePath {
		t.Errorf("go.mod module path mismatch:\n  declared:  %s\n  canonical: %s\n"+
			"A mismatch breaks `go install` for every consumer.", declared, canonicalModulePath)
	}
}
