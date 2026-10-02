package importer

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// writeFile is a helper to create a file with given content inside a base dir.
func writeFile(t *testing.T, base, rel, content string) {
	t.Helper()
	full := filepath.Join(base, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", rel, err)
	}
}

func TestDirHash_Deterministic(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "---\nversion: 1\n---\n# content")
	writeFile(t, dir, "README.md", "hello")

	h1, err := DirHash(dir)
	if err != nil {
		t.Fatalf("first DirHash: %v", err)
	}
	h2, err := DirHash(dir)
	if err != nil {
		t.Fatalf("second DirHash: %v", err)
	}
	if h1 != h2 {
		t.Errorf("DirHash not deterministic: %q != %q", h1, h2)
	}
}

func TestDirHash_OrderIndependent(t *testing.T) {
	// Two dirs with identical files but written in different order
	// must produce the same hash.
	dirA := t.TempDir()
	writeFile(t, dirA, "a.txt", "alpha")
	writeFile(t, dirA, "b.txt", "beta")
	writeFile(t, dirA, "sub/c.txt", "gamma")

	dirB := t.TempDir()
	writeFile(t, dirB, "sub/c.txt", "gamma")
	writeFile(t, dirB, "b.txt", "beta")
	writeFile(t, dirB, "a.txt", "alpha")

	hA, err := DirHash(dirA)
	if err != nil {
		t.Fatalf("DirHash dirA: %v", err)
	}
	hB, err := DirHash(dirB)
	if err != nil {
		t.Fatalf("DirHash dirB: %v", err)
	}
	if hA != hB {
		t.Errorf("DirHash not order-independent: %q != %q", hA, hB)
	}
}

func TestDirHash_ChangesOnContentChange(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "original")

	h1, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash before change: %v", err)
	}

	writeFile(t, dir, "SKILL.md", "modified")

	h2, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash after change: %v", err)
	}

	if h1 == h2 {
		t.Error("DirHash should change when file content changes")
	}
}

func TestDirHash_ChangesOnFileAdded(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "content")

	h1, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash before add: %v", err)
	}

	writeFile(t, dir, "extra.md", "new file")

	h2, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash after add: %v", err)
	}

	if h1 == h2 {
		t.Error("DirHash should change when a file is added")
	}
}

func TestDirHash_SkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}

	dir := t.TempDir()
	writeFile(t, dir, "SKILL.md", "content")

	// Create a symlink inside the dir; DirHash must not error and must
	// produce the same result as hashing without the symlink.
	h1, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash without symlink: %v", err)
	}

	symlinkPath := filepath.Join(dir, "link.md")
	target := filepath.Join(dir, "SKILL.md")
	if err := os.Symlink(target, symlinkPath); err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	h2, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash with symlink: %v", err)
	}

	if h1 != h2 {
		t.Error("DirHash should skip symlinks and produce same hash")
	}
}

func TestDirHash_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	h, err := DirHash(dir)
	if err != nil {
		t.Fatalf("DirHash empty dir: %v", err)
	}
	if h == "" {
		t.Error("DirHash should return a non-empty string for an empty dir")
	}
}
