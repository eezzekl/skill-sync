package discovery

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestScan(t *testing.T) {
	tmpDir := t.TempDir()

	cwdDir := filepath.Join(tmpDir, "cwd")
	homeDir := filepath.Join(tmpDir, "home")

	os.MkdirAll(cwdDir, 0755)
	os.MkdirAll(homeDir, 0755)

	tests := []struct {
		name         string
		setupFunc    func()
		expectedDirs []string
	}{
		{
			name:         "no directories exist",
			setupFunc:    func() {},
			expectedDirs: []string{},
		},
		{
			name: "some local and global directories exist",
			setupFunc: func() {
				// Local OpenCode
				os.MkdirAll(filepath.Join(cwdDir, ".opencode"), 0755)
				// Global Claude
				os.MkdirAll(filepath.Join(homeDir, ".claude"), 0755)
				// Local Cursor
				os.MkdirAll(filepath.Join(cwdDir, ".cursor"), 0755)
				// Global Pi
				os.MkdirAll(filepath.Join(homeDir, ".pi", "agent"), 0755)
			},
			expectedDirs: []string{
				filepath.Join(cwdDir, ".opencode"),
				filepath.Join(cwdDir, ".cursor"),
				filepath.Join(homeDir, ".claude"),
				filepath.Join(homeDir, ".pi", "agent"),
			},
		},
		{
			name: "all local and global directories exist",
			setupFunc: func() {
				// Local
				os.MkdirAll(filepath.Join(cwdDir, ".opencode"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".claude"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".cursor"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".gemini"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".antigravity"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".codex"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".codeium", "windsurf"), 0755)
				os.MkdirAll(filepath.Join(cwdDir, ".copilot"), 0755)

				// Global
				os.MkdirAll(filepath.Join(homeDir, ".config", "opencode"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".claude"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".cursor"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".gemini"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".gemini", "antigravity"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".codex"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".codeium", "windsurf"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".copilot"), 0755)
				os.MkdirAll(filepath.Join(homeDir, ".pi", "agent"), 0755)
			},
			expectedDirs: []string{
				filepath.Join(cwdDir, ".opencode"),
				filepath.Join(cwdDir, ".claude"),
				filepath.Join(cwdDir, ".cursor"),
				filepath.Join(cwdDir, ".gemini"),
				filepath.Join(cwdDir, ".antigravity"),
				filepath.Join(cwdDir, ".codex"),
				filepath.Join(cwdDir, ".codeium", "windsurf"),
				filepath.Join(cwdDir, ".copilot"),
				filepath.Join(homeDir, ".config", "opencode"),
				filepath.Join(homeDir, ".claude"),
				filepath.Join(homeDir, ".cursor"),
				filepath.Join(homeDir, ".gemini"),
				filepath.Join(homeDir, ".gemini", "antigravity"),
				filepath.Join(homeDir, ".codex"),
				filepath.Join(homeDir, ".codeium", "windsurf"),
				filepath.Join(homeDir, ".copilot"),
				filepath.Join(homeDir, ".pi", "agent"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear directories for isolation
			os.RemoveAll(cwdDir)
			os.RemoveAll(homeDir)
			os.MkdirAll(cwdDir, 0755)
			os.MkdirAll(homeDir, 0755)

			tt.setupFunc()

			d := &Discovery{
				Getwd: func() (string, error) {
					return cwdDir, nil
				},
				UserHomeDir: func() (string, error) {
					return homeDir, nil
				},
			}

			dirs, err := d.Scan()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Sort slices to make comparison stable
			sort.Strings(dirs)
			sort.Strings(tt.expectedDirs)

			if !reflect.DeepEqual(dirs, tt.expectedDirs) {
				t.Errorf("expected %v, got %v", tt.expectedDirs, dirs)
			}
		})
	}
}
