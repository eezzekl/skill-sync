package sync_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/sync"
)

func TestEngine_Sync(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, winnerPath string, targetDirs []string)
		winnerBody string
		targetDirs int
		wantErr    bool
	}{
		{
			name:       "syncs new file to multiple targets",
			winnerBody: "content version 1",
			targetDirs: 2,
			wantErr:    false,
			setup: func(t *testing.T, winnerPath string, targetDirs []string) {
				// Directories are empty
			},
		},
		{
			name:       "backs up existing file before overwrite",
			winnerBody: "content version 2",
			targetDirs: 1,
			wantErr:    false,
			setup: func(t *testing.T, winnerPath string, targetDirs []string) {
				// Pre-create the file in target dir under skills/
				dest := filepath.Join(targetDirs[0], "skills", "my-skill", "SKILL.md")
				err := os.MkdirAll(filepath.Dir(dest), 0755)
				if err != nil {
					t.Fatalf("failed to mkdir: %v", err)
				}
				os.WriteFile(dest, []byte("content version 1"), 0644)
			},
		},
		{
			name:       "skips missing base dirs but succeeds if one is valid",
			winnerBody: "content",
			targetDirs: 2,
			wantErr:    false,
			setup: func(t *testing.T, winnerPath string, targetDirs []string) {
				// targetDirs[0] is targetBase/A, targetDirs[1] is targetBase/B
				// We want targetDirs[0] to have a missing base dir.
				// We'll change targetDirs[0] to /nonexistent-base/A in the test loop below
			},
		},
		{
			name:       "returns error when all base dirs are missing",
			winnerBody: "content",
			targetDirs: 2,
			wantErr:    true,
			setup: func(t *testing.T, winnerPath string, targetDirs []string) {
				// Both have missing base dirs
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup winner
			winnerDir := t.TempDir()
			winnerPath := filepath.Join(winnerDir, "SKILL.md")
			os.WriteFile(winnerPath, []byte(tt.winnerBody), 0644)
			winner := models.SkillInstance{
				ID:   "my-skill",
				Path: winnerPath,
			}

			// Setup targets
			targetBase := t.TempDir()
			var targetDirs []string
			for i := 0; i < tt.targetDirs; i++ {
				dirPath := filepath.Join(targetBase, string(rune('A'+i)))

				if tt.name == "skips missing base dirs but succeeds if one is valid" && i == 0 {
					dirPath = filepath.Join(t.TempDir(), "nonexistent-base", string(rune('A'+i)))
				}
				if tt.name == "returns error when all base dirs are missing" {
					dirPath = filepath.Join(t.TempDir(), "nonexistent-base", string(rune('A'+i)))
				}

				targetDirs = append(targetDirs, dirPath)
			}

			if tt.setup != nil {
				tt.setup(t, winnerPath, targetDirs)
			}

			engine := sync.NewEngine()
			err := engine.Sync(winner, targetDirs)

			if (err != nil) != tt.wantErr {
				t.Fatalf("Sync() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				return
			}

			// Verify
			for _, target := range targetDirs {
				if filepath.Base(filepath.Dir(target)) == "nonexistent-base" {
					continue
				}
				dest := filepath.Join(target, "skills", "my-skill", "SKILL.md")
				b, err := os.ReadFile(dest)
				if err != nil {
					t.Errorf("failed to read dest: %v", err)
				}
				if string(b) != tt.winnerBody {
					t.Errorf("content = %q, want %q", string(b), tt.winnerBody)
				}

				// Check backup if this was an overwrite test
				if tt.name == "backs up existing file before overwrite" {
					bakPath := filepath.Join(target, "skills", "my-skill", "SKILL.md.bak")
					bak, err := os.ReadFile(bakPath)
					if err != nil {
						t.Errorf("failed to read backup: %v", err)
					}
					if string(bak) != "content version 1" {
						t.Errorf("backup content = %q, want %q", string(bak), "content version 1")
					}
				}
			}
		})
	}
}

// TestEngine_Sync_ErrorPaths covers OS-level error paths in engine.Sync().
//
// Notes on untestable paths without privileged operations:
//   - tempFile.Write() failure: requires a file descriptor that is closed or on a full filesystem.
//     Not injectable without OS-level fault injection on Windows/Linux without root.
//   - tempFile.Sync() failure: same constraint as Write().
//   - tempFile.Close() failure: same constraint as Write()/Sync() — OS buffers are already flushed
//     and close is infallible for normal files.
//
// These three paths are documented here and left untested to avoid brittle/privileged tests.
func TestEngine_Sync_ErrorPaths(t *testing.T) {
	t.Run("returns error when winner file does not exist", func(t *testing.T) {
		e := sync.NewEngine()
		winner := models.SkillInstance{
			ID:   "missing-skill",
			Path: filepath.Join(t.TempDir(), "nonexistent", "SKILL.md"),
		}
		err := e.Sync(winner, []string{t.TempDir()})
		if err == nil {
			t.Fatal("expected error when winner file does not exist, got nil")
		}
	})

	t.Run("returns error when MkdirAll fails (target path is a file)", func(t *testing.T) {
		// To force MkdirAll to fail, we create a regular file at the path that
		// MkdirAll would try to use as a directory component.
		base := t.TempDir()

		// Create winner
		winnerPath := filepath.Join(base, "SKILL.md")
		if err := os.WriteFile(winnerPath, []byte("content"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		winner := models.SkillInstance{ID: "my-skill", Path: winnerPath}

		// Place a regular file named "skills" so MkdirAll(skills/) fails
		targetDir := filepath.Join(base, "target")
		if err := os.MkdirAll(targetDir, 0755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		blockingFile := filepath.Join(targetDir, "skills")
		if err := os.WriteFile(blockingFile, []byte("blocker"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		e := sync.NewEngine()
		err := e.Sync(winner, []string{targetDir})
		if err == nil {
			t.Fatal("expected error when MkdirAll is blocked by a file, got nil")
		}
	})

	t.Run("returns error when os.Rename fails (target is read-only on Unix)", func(t *testing.T) {
		// On Windows, file-locking is done differently and this path is harder to trigger
		// without administrator privileges. We skip this test on Windows.
		if runtime.GOOS == "windows" {
			t.Skip("os.Rename failure via read-only dir is not reliably triggerable on Windows without admin")
		}

		base := t.TempDir()
		winnerPath := filepath.Join(base, "SKILL.md")
		if err := os.WriteFile(winnerPath, []byte("content"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		winner := models.SkillInstance{ID: "my-skill", Path: winnerPath}

		// Create the destDir under skills/, then make it read-only so rename fails
		destDir := filepath.Join(base, "target", "skills", "my-skill")
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		// Remove write permission so os.Rename (creating a file) will fail
		if err := os.Chmod(destDir, 0555); err != nil {
			t.Fatalf("setup chmod failed: %v", err)
		}
		// Restore permissions on cleanup so t.TempDir() cleanup works
		t.Cleanup(func() { os.Chmod(destDir, 0755) })

		e := sync.NewEngine()
		err := e.Sync(winner, []string{filepath.Join(base, "target")})
		if err == nil {
			t.Fatal("expected error when rename fails due to read-only directory, got nil")
		}
	})

	t.Run("returns error when backup copyFile fails (source is a directory)", func(t *testing.T) {
		// We force copyFile (the backup step) to fail by placing a DIRECTORY at
		// destPath so os.Open(destPath) succeeds (stat sees it exists) but os.Open
		// on a directory for copy-read will succeed while os.Create on dst can fail.
		// The easiest reliable approach: put a directory where SKILL.md.bak should go.
		//
		// Actually the simpler path: SKILL.md exists (triggers backup), then make
		// the target directory read-only so os.Create(bakPath) fails.
		// This is Unix-only (same constraint as Rename test).
		if runtime.GOOS == "windows" {
			t.Skip("backup failure via read-only dir is not reliably triggerable on Windows without admin")
		}

		base := t.TempDir()
		winnerPath := filepath.Join(base, "SKILL.md")
		if err := os.WriteFile(winnerPath, []byte("v2"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		winner := models.SkillInstance{ID: "my-skill", Path: winnerPath}

		// Pre-create target with an existing SKILL.md under skills/ so backup is triggered
		targetDir := filepath.Join(base, "target")
		skillDir := filepath.Join(targetDir, "skills", "my-skill")
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("v1"), 0644); err != nil {
			t.Fatalf("setup failed: %v", err)
		}
		// Make skill dir read-only so backup creation (os.Create) will fail
		if err := os.Chmod(skillDir, 0555); err != nil {
			t.Fatalf("setup chmod failed: %v", err)
		}
		t.Cleanup(func() { os.Chmod(skillDir, 0755) })

		e := sync.NewEngine()
		err := e.Sync(winner, []string{targetDir})
		if err == nil {
			t.Fatal("expected error when backup creation fails due to read-only directory, got nil")
		}
	})
}
