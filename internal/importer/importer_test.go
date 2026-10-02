package importer

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

	"github.com/eezzekl/skill-sync/internal/models"
)

// makeSkillDir creates a minimal skill directory with a SKILL.md at
// <targetDir>/skills/<skillID>/SKILL.md. Returns the skill dir path.
func makeSkillDir(t *testing.T, targetDir, skillID, frontmatter string, extraFiles map[string]string) string {
	t.Helper()
	skillDir := filepath.Join(targetDir, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", skillDir, err)
	}
	mdContent := "---\n" + frontmatter + "\n---\n# " + skillID
	writeFile(t, skillDir, "SKILL.md", mdContent)
	for rel, content := range extraFiles {
		writeFile(t, skillDir, rel, content)
	}
	return skillDir
}

// ── FindSkill ────────────────────────────────────────────────────────────────

func TestFindSkill_ReturnsFromMultipleTargets(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	makeSkillDir(t, t1, "git-expert", "version: 1\nname: Git Expert\n", nil)
	makeSkillDir(t, t2, "git-expert", "version: 2\nname: Git Expert\n", nil)

	imp := New([]string{t1, t2}, t.TempDir())
	candidates, err := imp.FindSkill("git-expert")
	if err != nil {
		t.Fatalf("FindSkill: %v", err)
	}
	if len(candidates) != 2 {
		t.Errorf("got %d candidates, want 2", len(candidates))
	}
}

func TestFindSkill_ReturnsEmptyForUnknownID(t *testing.T) {
	target := t.TempDir()
	makeSkillDir(t, target, "docker-dev", "version: 1\n", nil)

	imp := New([]string{target}, t.TempDir())
	candidates, err := imp.FindSkill("nonexistent")
	if err != nil {
		t.Fatalf("FindSkill: %v", err)
	}
	if len(candidates) != 0 {
		t.Errorf("got %d candidates, want 0", len(candidates))
	}
}

func TestFindSkill_SkipsMissingSkillsDir(t *testing.T) {
	existing := t.TempDir()
	missing := t.TempDir() // create dir but don't populate skills/ subdir
	makeSkillDir(t, existing, "my-skill", "version: 1\n", nil)

	imp := New([]string{existing, missing}, t.TempDir())
	candidates, err := imp.FindSkill("my-skill")
	if err != nil {
		t.Fatalf("FindSkill: %v", err)
	}
	if len(candidates) != 1 {
		t.Errorf("got %d candidates, want 1 (missing target should be skipped)", len(candidates))
	}
}

func TestFindSkill_ReadsFrontmatter(t *testing.T) {
	target := t.TempDir()
	makeSkillDir(t, target, "k8s-ops", "version: 1.5\nname: K8s Ops\ndescription: Kubernetes skill\n", nil)

	imp := New([]string{target}, t.TempDir())
	candidates, err := imp.FindSkill("k8s-ops")
	if err != nil {
		t.Fatalf("FindSkill: %v", err)
	}
	if len(candidates) != 1 {
		t.Fatalf("got %d candidates, want 1", len(candidates))
	}
	meta := candidates[0].Metadata
	if meta.Version != 1.5 {
		t.Errorf("Version: got %v, want 1.5", meta.Version)
	}
	if meta.Name != "K8s Ops" {
		t.Errorf("Name: got %q, want %q", meta.Name, "K8s Ops")
	}
	if meta.Description != "Kubernetes skill" {
		t.Errorf("Description: got %q, want %q", meta.Description, "Kubernetes skill")
	}
}

// ── Resolve ──────────────────────────────────────────────────────────────────

func TestResolve_HighestVersionWins(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	makeSkillDir(t, t1, "my-skill", "version: 1\n", nil)
	makeSkillDir(t, t2, "my-skill", "version: 2\n", nil)

	imp := New([]string{t1, t2}, t.TempDir())
	candidates, _ := imp.FindSkill("my-skill")

	winner, conflict, err := Resolve(candidates)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if conflict {
		t.Fatal("unexpected conflict")
	}
	if winner.Metadata.Version != 2 {
		t.Errorf("winner version: got %v, want 2", winner.Metadata.Version)
	}
}

func TestResolve_NewestMtimeWinsOnVersionTie(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	makeSkillDir(t, t1, "my-skill", "version: 1\nname: old\n", nil)
	makeSkillDir(t, t2, "my-skill", "version: 1\nname: new\n", nil)

	// Force t2's SKILL.md to have a newer mtime.
	mdPath := filepath.Join(t2, "skills", "my-skill", "SKILL.md")
	future := time.Now().Add(time.Hour)
	if err := os.Chtimes(mdPath, future, future); err != nil {
		t.Fatalf("Chtimes: %v", err)
	}

	imp := New([]string{t1, t2}, t.TempDir())
	candidates, _ := imp.FindSkill("my-skill")

	winner, conflict, err := Resolve(candidates)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if conflict {
		t.Fatal("unexpected conflict")
	}
	if winner.AgentDir != t2 {
		t.Errorf("winner AgentDir: got %q, want %q", winner.AgentDir, t2)
	}
}

func TestResolve_ConflictOnTie(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	// Same version, same mtime (we'll force both), different content → true tie.
	makeSkillDir(t, t1, "my-skill", "version: 1\nname: alpha\n", nil)
	makeSkillDir(t, t2, "my-skill", "version: 1\nname: beta\n", nil)

	// Force identical mtime on both SKILL.md files.
	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, base := range []string{t1, t2} {
		mdPath := filepath.Join(base, "skills", "my-skill", "SKILL.md")
		if err := os.Chtimes(mdPath, fixed, fixed); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
	}

	imp := New([]string{t1, t2}, t.TempDir())
	candidates, _ := imp.FindSkill("my-skill")

	_, conflict, err := Resolve(candidates)
	if !conflict {
		t.Error("expected conflict=true")
	}
	if !errors.Is(err, ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestResolve_ErrSkillNotFoundOnEmpty(t *testing.T) {
	_, _, err := Resolve(nil)
	if !errors.Is(err, ErrSkillNotFound) {
		t.Errorf("expected ErrSkillNotFound, got %v", err)
	}
}

// ── FindAllSkills ─────────────────────────────────────────────────────────────

func TestFindAllSkills_ReturnsOneEntryPerSkillID(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	makeSkillDir(t, t1, "git-expert", "version: 2\n", nil)
	makeSkillDir(t, t2, "git-expert", "version: 1\n", nil)
	makeSkillDir(t, t1, "docker-dev", "version: 1\n", nil)

	imp := New([]string{t1, t2}, t.TempDir())
	results, err := imp.FindAllSkills()
	if err != nil {
		t.Fatalf("FindAllSkills: %v", err)
	}
	if len(results) != 2 {
		t.Errorf("got %d results, want 2", len(results))
	}
}

func TestFindAllSkills_MarksConflict(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()
	// Different content, same version, same forced mtime → conflict.
	makeSkillDir(t, t1, "my-skill", "version: 1\nname: alpha\n", nil)
	makeSkillDir(t, t2, "my-skill", "version: 1\nname: beta\n", nil)

	fixed := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, base := range []string{t1, t2} {
		mdPath := filepath.Join(base, "skills", "my-skill", "SKILL.md")
		if err := os.Chtimes(mdPath, fixed, fixed); err != nil {
			t.Fatalf("Chtimes: %v", err)
		}
	}

	imp := New([]string{t1, t2}, t.TempDir())
	results, err := imp.FindAllSkills()
	if err != nil {
		t.Fatalf("FindAllSkills: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
	if !results[0].IsConflict {
		t.Error("expected IsConflict=true")
	}
}

func TestFindAllSkills_SortedBySkillID(t *testing.T) {
	target := t.TempDir()
	makeSkillDir(t, target, "zebra", "version: 1\n", nil)
	makeSkillDir(t, target, "alpha", "version: 1\n", nil)
	makeSkillDir(t, target, "mango", "version: 1\n", nil)

	imp := New([]string{target}, t.TempDir())
	results, err := imp.FindAllSkills()
	if err != nil {
		t.Fatalf("FindAllSkills: %v", err)
	}

	ids := make([]string, len(results))
	for i, r := range results {
		ids[i] = r.SkillID
	}
	if !sort.StringsAreSorted(ids) {
		t.Errorf("results not sorted by SkillID: %v", ids)
	}
}

func TestFindAllSkills_SkipsMissingSkillsDir(t *testing.T) {
	existing := t.TempDir()
	empty := t.TempDir()
	makeSkillDir(t, existing, "my-skill", "version: 1\n", nil)

	imp := New([]string{existing, empty}, t.TempDir())
	results, err := imp.FindAllSkills()
	if err != nil {
		t.Fatalf("FindAllSkills: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("got %d, want 1", len(results))
	}
}

// ── CopySkill ─────────────────────────────────────────────────────────────────

func TestCopySkill_CopiesFullTree(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "SKILL.md", "---\nversion: 1\n---")
	writeFile(t, src, "examples/demo.sh", "#!/bin/bash")
	writeFile(t, src, "docs/README.md", "# Docs")

	dest := t.TempDir()
	imp := New(nil, dest)
	if err := imp.CopySkill("my-skill", src); err != nil {
		t.Fatalf("CopySkill: %v", err)
	}

	for _, rel := range []string{"SKILL.md", "examples/demo.sh", "docs/README.md"} {
		p := filepath.Join(dest, "my-skill", rel)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected file %s to exist: %v", rel, err)
		}
	}
}

func TestCopySkill_CreatesBakOnOverwrite(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "SKILL.md", "new content")

	dest := t.TempDir()
	// Pre-populate the destination.
	writeFile(t, dest, "my-skill/SKILL.md", "old content")

	imp := New(nil, dest)
	if err := imp.CopySkill("my-skill", src); err != nil {
		t.Fatalf("CopySkill: %v", err)
	}

	bakPath := filepath.Join(dest, "my-skill.bak", "SKILL.md")
	if _, err := os.Stat(bakPath); err != nil {
		t.Errorf("expected backup at %s: %v", bakPath, err)
	}
}

func TestCopySkill_RemovesStaleBak(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "SKILL.md", "content")

	dest := t.TempDir()
	writeFile(t, dest, "my-skill/SKILL.md", "existing")
	// Pre-create a stale .bak
	writeFile(t, dest, "my-skill.bak/OLD.md", "stale")

	imp := New(nil, dest)
	if err := imp.CopySkill("my-skill", src); err != nil {
		t.Fatalf("CopySkill: %v", err)
	}

	// Stale file must be gone; new .bak should have been created from existing.
	oldFile := filepath.Join(dest, "my-skill.bak", "OLD.md")
	if _, err := os.Stat(oldFile); err == nil {
		t.Error("stale .bak file should have been removed")
	}

	newBak := filepath.Join(dest, "my-skill.bak", "SKILL.md")
	if _, err := os.Stat(newBak); err != nil {
		t.Errorf("expected new .bak at %s: %v", newBak, err)
	}
}

func TestCopySkill_Idempotent(t *testing.T) {
	src := t.TempDir()
	writeFile(t, src, "SKILL.md", "---\nversion: 1\n---\n# skill")

	dest := t.TempDir()
	imp := New(nil, dest)

	if err := imp.CopySkill("my-skill", src); err != nil {
		t.Fatalf("first CopySkill: %v", err)
	}
	if err := imp.CopySkill("my-skill", src); err != nil {
		t.Fatalf("second CopySkill: %v", err)
	}

	// Content must be the same after two identical copies.
	got, err := os.ReadFile(filepath.Join(dest, "my-skill", "SKILL.md"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != "---\nversion: 1\n---\n# skill" {
		t.Errorf("unexpected content after idempotent copy: %q", got)
	}
}

func TestCopySkill_LeavesExistingUntouchedOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only file injection is unreliable on Windows")
	}

	src := t.TempDir()
	writeFile(t, src, "SKILL.md", "new")
	writeFile(t, src, "extra.md", "extra")

	dest := t.TempDir()
	// Pre-populate the destination with known content.
	writeFile(t, dest, "my-skill/SKILL.md", "original")

	// Make the destination skills root read-only so .tmp creation fails.
	destRoot := filepath.Join(dest)
	if err := os.Chmod(destRoot, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(destRoot, 0o755) })

	imp := New(nil, dest)
	err := imp.CopySkill("my-skill", src)
	if err == nil {
		t.Fatal("expected error when dest root is read-only")
	}

	// Original must be untouched.
	original, readErr := os.ReadFile(filepath.Join(dest, "my-skill", "SKILL.md"))
	if readErr != nil {
		t.Fatalf("ReadFile original: %v", readErr)
	}
	if string(original) != "original" {
		t.Errorf("original content changed: %q", original)
	}
}

// ── readMetadata ─────────────────────────────────────────────────────────────

func TestReadMetadata(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    models.SkillMetadata
	}{
		{
			name:    "valid frontmatter",
			content: "---\nversion: 1.5\nname: My Skill\ndescription: A skill\n---\n# body",
			want:    models.SkillMetadata{Version: 1.5, Name: "My Skill", Description: "A skill"},
		},
		{
			name:    "no frontmatter",
			content: "# Just a heading",
			want:    models.SkillMetadata{},
		},
		{
			name:    "unclosed frontmatter",
			content: "---\nversion: 1\n",
			want:    models.SkillMetadata{},
		},
		{
			name:    "missing fields default to zero",
			content: "---\nversion: 2\n---\n# body",
			want:    models.SkillMetadata{Version: 2},
		},
		{
			name:    "crlf line endings",
			content: "---\r\nversion: 1\r\nname: CRLF Skill\r\n---\r\n# body",
			want:    models.SkillMetadata{Version: 1, Name: "CRLF Skill"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			mdPath := filepath.Join(dir, "SKILL.md")
			if err := os.WriteFile(mdPath, []byte(tt.content), 0o644); err != nil {
				t.Fatalf("WriteFile: %v", err)
			}
			got, _ := readMetadata(mdPath)
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
