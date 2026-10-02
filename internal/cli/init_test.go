package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitCmd(t *testing.T) {
	origGetWd := getWd
	origGetUserConfig := getUserConfig
	origGetUserHome := getUserHome
	origIsTTY := isTTY
	origStdin := stdinReader
	defer func() {
		getWd = origGetWd
		getUserConfig = origGetUserConfig
		getUserHome = origGetUserHome
		isTTY = origIsTTY
		stdinReader = origStdin
	}()

	// Pin the home seam to an empty directory so these cases never discover the
	// developer's real agent directories. Global discovery has dedicated
	// coverage in TestInitDiscoversGlobalAgents.
	emptyHome := t.TempDir()
	getUserHome = func() (string, error) { return emptyHome, nil }

	t.Run("local init creates file in cwd with skills suffix", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return false }

		os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755)

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := filepath.Join(tempDir, "skill-sync.yaml")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("expected config file at %s", expectedPath)
		}

		content, _ := os.ReadFile(expectedPath)
		if !strings.Contains(string(content), ".claude/skills") {
			t.Errorf("expected config to contain '.claude/skills', got: %s", string(content))
		}
	})

	t.Run("global init creates file in user config dir", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		userDir := filepath.Join(tempDir, "user_config")
		getUserConfig = func() (string, error) { return userDir, nil }
		isTTY = func() bool { return false }

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedPath := filepath.Join(userDir, "skill-sync", "skill-sync.yaml")
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			t.Errorf("expected config file at %s", expectedPath)
		}
	})

	t.Run("no agents found prints 'no skills configured'", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return false }

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "no skills configured") {
			t.Errorf("expected 'no skills configured', got: %s", out.String())
		}
	})

	t.Run("non-TTY aborts with error when config exists", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return false }

		os.WriteFile(filepath.Join(tempDir, "skill-sync.yaml"), []byte("old config"), 0644)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--local"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error on non-TTY with existing config and no --force")
		}
		if !strings.Contains(err.Error(), "already exists") {
			t.Errorf("expected 'already exists' in error, got %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(tempDir, "skill-sync.yaml"))
		if string(content) != "old config" {
			t.Error("expected file to be unchanged")
		}
	})

	t.Run("TTY prompts and proceeds on 'y'", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return true }
		stdinReader = strings.NewReader("y\n")

		os.WriteFile(filepath.Join(tempDir, "skill-sync.yaml"), []byte("old config"), 0644)

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "reinitialize?") {
			t.Errorf("expected prompt in output, got: %s", out.String())
		}
		content, _ := os.ReadFile(filepath.Join(tempDir, "skill-sync.yaml"))
		if string(content) == "old config" {
			t.Error("expected config to be overwritten after answering 'y'")
		}
	})

	t.Run("TTY prompts and aborts on non-y input", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return true }
		stdinReader = strings.NewReader("\n") // empty = default N

		os.WriteFile(filepath.Join(tempDir, "skill-sync.yaml"), []byte("old config"), 0644)

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if !strings.Contains(out.String(), "aborted") {
			t.Errorf("expected 'aborted' in output, got: %s", out.String())
		}
		content, _ := os.ReadFile(filepath.Join(tempDir, "skill-sync.yaml"))
		if string(content) != "old config" {
			t.Error("expected file to be unchanged after declining prompt")
		}
	})

	t.Run("overwrites if config exists with --force", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }
		isTTY = func() bool { return false }

		os.WriteFile(filepath.Join(tempDir, "skill-sync.yaml"), []byte("old config"), 0644)

		cmd := NewInitCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local", "--force"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error with --force: %v", err)
		}

		content, _ := os.ReadFile(filepath.Join(tempDir, "skill-sync.yaml"))
		if string(content) == "old config" {
			t.Error("expected config to be overwritten, but found old content")
		}
	})
}

// TestInitDiscoversGlobalAgents pins the home/config seam separation. Agent
// directories live under the user home, not under the user config directory, so
// init must keep those two lookups independent. Pi is the canary: it has no
// local directory, so a collapsed seam makes it undiscoverable.
func TestInitDiscoversGlobalAgents(t *testing.T) {
	origGetWd := getWd
	origGetUserConfig := getUserConfig
	origGetUserHome := getUserHome
	origIsTTY := isTTY
	defer func() {
		getWd = origGetWd
		getUserConfig = origGetUserConfig
		getUserHome = origGetUserHome
		isTTY = origIsTTY
	}()

	tests := []struct {
		name      string
		globalDir string
		want      string
	}{
		{
			name:      "discovers Pi which is global only",
			globalDir: filepath.Join(".pi", "agent"),
			want:      ".pi/agent/skills",
		},
		{
			name:      "discovers global Claude Code",
			globalDir: ".claude",
			want:      ".claude/skills",
		},
		{
			name:      "discovers global OpenCode under .config",
			globalDir: filepath.Join(".config", "opencode"),
			want:      ".config/opencode/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			homeDir := filepath.Join(root, "home")
			// The config directory is deliberately a sibling of home, never its
			// parent, so a seam that confuses the two cannot accidentally pass.
			configDir := filepath.Join(root, "xdg_config")
			cwd := filepath.Join(root, "project")

			if err := os.MkdirAll(filepath.Join(homeDir, tt.globalDir), 0755); err != nil {
				t.Fatalf("failed to seed agent dir: %v", err)
			}
			if err := os.MkdirAll(cwd, 0755); err != nil {
				t.Fatalf("failed to seed cwd: %v", err)
			}

			getWd = func() (string, error) { return cwd, nil }
			getUserConfig = func() (string, error) { return configDir, nil }
			getUserHome = func() (string, error) { return homeDir, nil }
			isTTY = func() bool { return false }

			cmd := NewInitCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(filepath.Join(configDir, "skill-sync", "skill-sync.yaml"))
			if err != nil {
				t.Fatalf("failed to read generated config: %v", err)
			}

			if !strings.Contains(string(content), tt.want) {
				t.Errorf("expected config to contain %q, got:\n%s", tt.want, string(content))
			}
		})
	}
}
