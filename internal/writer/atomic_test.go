package writer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAtomicWrite(t *testing.T) {
	t.Run("creates new file atomically when none exists", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "target.txt")
		content := []byte("hello world")

		err := AtomicWrite(destPath, content)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify content
		got, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(got) != string(content) {
			t.Errorf("expected %q, got %q", content, got)
		}

		// Verify no backup was created
		_, err = os.Stat(destPath + ".bak")
		if !os.IsNotExist(err) {
			t.Errorf("expected backup file to not exist")
		}

		// Verify temp files are cleaned up
		files, _ := os.ReadDir(dir)
		if len(files) != 1 {
			t.Errorf("expected exactly 1 file in directory, got %d", len(files))
		}
	})

	t.Run("creates backup and overwrites when file exists", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "target.txt")
		oldContent := []byte("old content")
		newContent := []byte("new content")

		// Pre-create file
		if err := os.WriteFile(destPath, oldContent, 0644); err != nil {
			t.Fatalf("failed to setup initial file: %v", err)
		}

		err := AtomicWrite(destPath, newContent)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		// Verify new content
		got, err := os.ReadFile(destPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(got) != string(newContent) {
			t.Errorf("expected %q, got %q", newContent, got)
		}

		// Verify backup content
		bakGot, err := os.ReadFile(destPath + ".bak")
		if err != nil {
			t.Fatalf("failed to read backup file: %v", err)
		}
		if string(bakGot) != string(oldContent) {
			t.Errorf("expected backup %q, got %q", oldContent, bakGot)
		}
	})

	t.Run("returns error when target directory does not exist", func(t *testing.T) {
		dir := t.TempDir()
		destPath := filepath.Join(dir, "nonexistent", "target.txt")
		
		err := AtomicWrite(destPath, []byte("data"))
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		
		if !strings.Contains(err.Error(), "no such file or directory") && !strings.Contains(err.Error(), "The system cannot find the path specified") {
			t.Errorf("expected path error, got %v", err)
		}
	})
}