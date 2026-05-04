package config_resolver

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve(t *testing.T) {
	// Setup temp directories
	tmpDir := t.TempDir()

	cwdDir := filepath.Join(tmpDir, "cwd")
	userCfgDir := filepath.Join(tmpDir, "user_config")
	flagDir := filepath.Join(tmpDir, "flag")

	os.MkdirAll(cwdDir, 0755)
	os.MkdirAll(userCfgDir, 0755)
	os.MkdirAll(flagDir, 0755)

	// Create config files
	flagFile := filepath.Join(flagDir, "custom.yaml")
	os.WriteFile(flagFile, []byte(""), 0644)

	localCfgFile := filepath.Join(cwdDir, "skill-sync.yaml")
	os.WriteFile(localCfgFile, []byte(""), 0644)

	globalDir := filepath.Join(userCfgDir, "skill-sync")
	os.MkdirAll(globalDir, 0755)
	globalCfgFile := filepath.Join(globalDir, "skill-sync.yaml")
	os.WriteFile(globalCfgFile, []byte(""), 0644)

	tests := []struct {
		name         string
		flagPath     string
		setupLocal   bool
		setupGlobal  bool
		expectedPath string
		expectedErr  bool
		expectErrOut string
	}{
		{
			name:         "flag precedence over local and global",
			flagPath:     flagFile,
			setupLocal:   true,
			setupGlobal:  true,
			expectedPath: flagFile,
		},
		{
			name:         "local precedence over global",
			flagPath:     "",
			setupLocal:   true,
			setupGlobal:  true,
			expectedPath: localCfgFile,
			expectErrOut: "using local config " + localCfgFile,
		},
		{
			name:         "global fallback when no local or flag",
			flagPath:     "",
			setupLocal:   false,
			setupGlobal:  true,
			expectedPath: globalCfgFile,
		},
		{
			name:         "error when config not found anywhere",
			flagPath:     "",
			setupLocal:   false,
			setupGlobal:  false,
			expectedPath: "",
			expectedErr:  true,
		},
		{
			name:         "error when flag path does not exist",
			flagPath:     filepath.Join(flagDir, "nonexistent.yaml"),
			setupLocal:   true,
			setupGlobal:  true,
			expectedPath: "",
			expectedErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stderr bytes.Buffer

			// Setup functions to return our test directories based on test case
			r := &Resolver{
				Getwd: func() (string, error) {
					if tt.setupLocal {
						return cwdDir, nil
					}
					return filepath.Join(tmpDir, "empty_cwd"), nil
				},
				UserConfigDir: func() (string, error) {
					if tt.setupGlobal {
						return userCfgDir, nil
					}
					return filepath.Join(tmpDir, "empty_user_config"), nil
				},
				Stderr: &stderr,
			}

			// Ensure empty dirs exist if needed
			os.MkdirAll(filepath.Join(tmpDir, "empty_cwd"), 0755)
			os.MkdirAll(filepath.Join(tmpDir, "empty_user_config"), 0755)

			path, err := r.Resolve(tt.flagPath)

			if tt.expectedErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if path != tt.expectedPath {
				t.Errorf("expected path %q, got %q", tt.expectedPath, path)
			}

			outStr := stderr.String()
			if tt.expectErrOut != "" && !strings.Contains(outStr, tt.expectErrOut) {
				t.Errorf("expected stderr to contain %q, got %q", tt.expectErrOut, outStr)
			}
			if tt.expectErrOut == "" && outStr != "" {
				t.Errorf("expected empty stderr, got %q", outStr)
			}
		})
	}
}
