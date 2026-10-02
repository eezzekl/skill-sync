package importer

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

// DirHash computes a deterministic, cross-platform SHA-256 over all regular
// files in dir. Symlinks are skipped.
//
// Algorithm:
//  1. Walk dir recursively; collect relative paths of all regular files.
//  2. Normalize paths to forward slashes (cross-platform determinism).
//  3. Sort paths lexicographically.
//  4. For each path: feed "<normalizedRelPath>\n" + file bytes into the hasher.
//  5. Return hex-encoded SHA-256 of the accumulated digest.
func DirHash(dir string) (string, error) {
	type entry struct {
		rel     string
		absPath string
	}
	var entries []entry

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip symlinks and directories; only hash regular files.
		if d.Type()&fs.ModeSymlink != 0 || d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		entries = append(entries, entry{
			rel:     filepath.ToSlash(rel), // cross-platform path normalisation
			absPath: path,
		})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("DirHash walk %s: %w", dir, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	h := sha256.New()
	for _, e := range entries {
		data, err := os.ReadFile(e.absPath)
		if err != nil {
			return "", fmt.Errorf("DirHash read %s: %w", e.rel, err)
		}
		h.Write([]byte(e.rel + "\n"))
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
