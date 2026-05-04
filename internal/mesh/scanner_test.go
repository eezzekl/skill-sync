package mesh_test

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	"github.com/ezzek/skill-sync/internal/mesh"
)

func TestScanner_Scan(t *testing.T) {
	// Setup mock filesystem
	tempDir := t.TempDir()

	targetA := filepath.Join(tempDir, "targetA")
	targetB := filepath.Join(tempDir, "targetB")

	os.MkdirAll(filepath.Join(targetA, "skills", "skill-1"), 0755)
	os.MkdirAll(filepath.Join(targetB, "skills", "skill-1"), 0755)
	os.MkdirAll(filepath.Join(targetA, "skills", "skill-2"), 0755)

	content1A := []byte(`---
version: 1
name: Skill One A
---
# Skill One
Body A`)

	content1B := []byte(`---
version: 2
name: Skill One B
---
# Skill One
Body B`)

	content2A := []byte(`---
version: 1
name: Skill Two
---
# Skill Two
Body`)

	os.WriteFile(filepath.Join(targetA, "skills", "skill-1", "SKILL.md"), content1A, 0644)
	os.WriteFile(filepath.Join(targetB, "skills", "skill-1", "SKILL.md"), content1B, 0644)
	os.WriteFile(filepath.Join(targetA, "skills", "skill-2", "SKILL.md"), content2A, 0644)

	// Additional non-skill files that should be ignored
	os.WriteFile(filepath.Join(targetA, "skills", "skill-1", "OTHER.md"), []byte("Ignore me"), 0644)
	os.MkdirAll(filepath.Join(targetB, "skills", "empty-dir"), 0755)

	scanner := mesh.NewDefaultScanner()

	instances, err := scanner.Scan([]string{targetA, targetB})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(instances) != 2 {
		t.Fatalf("expected 2 unique skills, got %d", len(instances))
	}

	skill1Instances := instances["skill-1"]
	if len(skill1Instances) != 2 {
		t.Fatalf("expected 2 instances of skill-1, got %d", len(skill1Instances))
	}

	skill2Instances := instances["skill-2"]
	if len(skill2Instances) != 1 {
		t.Fatalf("expected 1 instance of skill-2, got %d", len(skill2Instances))
	}

	// Verify details of targetA/skill-1
	var inst1AFound bool
	for _, inst := range skill1Instances {
		if inst.TargetDir == targetA {
			inst1AFound = true
			if inst.ID != "skill-1" {
				t.Errorf("expected ID 'skill-1', got %q", inst.ID)
			}
			if inst.Metadata.Version != 1 {
				t.Errorf("expected Version 1, got %g", inst.Metadata.Version)
			}
			if inst.Metadata.Name != "Skill One A" {
				t.Errorf("expected Name 'Skill One A', got %q", inst.Metadata.Name)
			}
			expectedHash := sha256Sum(content1A)
			if inst.Hash != expectedHash {
				t.Errorf("expected Hash %q, got %q", expectedHash, inst.Hash)
			}
			if inst.Mtime.IsZero() {
				t.Errorf("expected non-zero Mtime")
			}
		}
	}
	if !inst1AFound {
		t.Errorf("did not find targetA/skill-1 instance")
	}
}

func sha256Sum(b []byte) string {
	h := sha256.New()
	h.Write(b)
	return hex.EncodeToString(h.Sum(nil))
}
