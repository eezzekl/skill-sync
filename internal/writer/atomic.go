package writer

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AtomicWrite writes content to destPath atomically.
// If destPath already exists, it creates a .bak backup file before overwriting.
func AtomicWrite(destPath string, content []byte) error {
	destDir := filepath.Dir(destPath)

	// Backup if exists
	if _, err := os.Stat(destPath); err == nil {
		bakPath := destPath + ".bak"
		// Create a copy for the backup
		if err := copyFile(destPath, bakPath); err != nil {
			return fmt.Errorf("failed to backup existing file: %w", err)
		}
	}

	// Atomic write
	tempFile, err := os.CreateTemp(destDir, "atomic-tmp-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tempName := tempFile.Name()

	// Ensure cleanup in case of panic or early return
	defer func() {
		if tempFile != nil {
			tempFile.Close()
		}
		os.Remove(tempName) // Attempt to remove temp file if still present
	}()

	if _, err := tempFile.Write(content); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := tempFile.Sync(); err != nil {
		return fmt.Errorf("failed to sync temp file: %w", err)
	}

	if err := tempFile.Close(); err != nil {
		tempFile = nil // Prevent defer from closing again
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	tempFile = nil // Closed successfully

	// Rename over the destination path
	if err := os.Rename(tempName, destPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
