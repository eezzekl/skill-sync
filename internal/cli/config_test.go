package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConfigCmd(t *testing.T) {
	origGetWd := getWd
	origGetUserConfig := getUserConfig
	origGetUserHome := getUserHome
	defer func() {
		getWd = origGetWd
		getUserConfig = origGetUserConfig
		getUserHome = origGetUserHome
	}()

	// Pin the home seam to an empty directory so these cases never discover the
	// developer's real agent directories. Global discovery has dedicated
	// coverage in TestConfigDiscoversGlobalAgents.
	emptyHome := t.TempDir()
	getUserHome = func() (string, error) { return emptyHome, nil }

	t.Run("default targets global config", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		userDir := filepath.Join(tempDir, "user_config")
		getUserConfig = func() (string, error) { return userDir, nil }

		cfgPath := filepath.Join(userDir, "skill-sync", "skill-sync.yaml")
		os.MkdirAll(filepath.Dir(cfgPath), 0755)
		os.WriteFile(cfgPath, []byte("targets:\n  - old\n"), 0644)

		// Put a local config too — must NOT be picked up
		os.WriteFile(filepath.Join(tempDir, "skill-sync.yaml"), []byte("targets:\n  - local\n"), 0644)

		cmd := NewConfigCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{}) // no flags → global

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(cfgPath)
		if strings.Contains(string(content), "old") {
			t.Error("expected global config to be updated, still has old content")
		}
	})

	t.Run("--local targets cwd config", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		userDir := filepath.Join(tempDir, "user_config")
		getUserConfig = func() (string, error) { return userDir, nil }

		localCfg := filepath.Join(tempDir, "skill-sync.yaml")
		os.WriteFile(localCfg, []byte("targets:\n  - old\n"), 0644)

		// Put a global config too — must NOT be touched
		globalCfg := filepath.Join(userDir, "skill-sync", "skill-sync.yaml")
		os.MkdirAll(filepath.Dir(globalCfg), 0755)
		os.WriteFile(globalCfg, []byte("targets:\n  - global-original\n"), 0644)

		cmd := NewConfigCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Local config must be updated
		content, _ := os.ReadFile(localCfg)
		if strings.Contains(string(content), "old") {
			t.Error("expected local config to be updated, still has old content")
		}

		// Global config must be untouched
		globalContent, _ := os.ReadFile(globalCfg)
		if !strings.Contains(string(globalContent), "global-original") {
			t.Error("expected global config to be untouched")
		}
	})

	t.Run("--config uses explicit path", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }

		cfgPath := filepath.Join(tempDir, "custom.yaml")
		os.WriteFile(cfgPath, []byte("targets:\n  - old\n"), 0644)

		cmd := NewConfigCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--config", cfgPath})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(cfgPath)
		if strings.Contains(string(content), "old") {
			t.Error("expected config to be updated via explicit path")
		}
	})

	t.Run("creates .bak backup before overwriting", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }

		cfgPath := filepath.Join(tempDir, "skill-sync.yaml")
		os.WriteFile(cfgPath, []byte("targets:\n  - original\n"), 0644)

		cmd := NewConfigCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		bak, err := os.ReadFile(cfgPath + ".bak")
		if err != nil {
			t.Fatalf("expected .bak file to exist: %v", err)
		}
		if !strings.Contains(string(bak), "original") {
			t.Errorf("expected .bak to contain original content, got: %s", string(bak))
		}
	})

	t.Run("preserves other yaml keys when updating targets", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }

		os.MkdirAll(filepath.Join(tempDir, ".claude"), 0755)

		cfgPath := filepath.Join(tempDir, "skill-sync.yaml")
		os.WriteFile(cfgPath, []byte("foo: bar\ntargets:\n  - old-target\n"), 0644)

		cmd := NewConfigCmd()
		var out bytes.Buffer
		cmd.SetOut(&out)
		cmd.SetArgs([]string{"--local"})

		if err := cmd.Execute(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, _ := os.ReadFile(cfgPath)
		strContent := string(content)
		if !strings.Contains(strContent, "foo: bar") {
			t.Errorf("expected 'foo: bar' to be preserved, got:\n%s", strContent)
		}
		if strings.Contains(strContent, "old-target") {
			t.Errorf("expected old-target to be replaced, got:\n%s", strContent)
		}
		if !strings.Contains(strContent, ".claude/skills") {
			t.Errorf("expected '.claude/skills' in updated targets, got:\n%s", strContent)
		}
	})

	t.Run("errors when config file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		getWd = func() (string, error) { return tempDir, nil }
		getUserConfig = func() (string, error) { return tempDir, nil }

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"--local"})

		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error when config file does not exist")
		}
		if !strings.Contains(err.Error(), "skill-sync init") {
			t.Errorf("expected hint to run 'skill-sync init', got: %v", err)
		}
	})
}

// TestConfigDiscoversGlobalAgents mirrors TestInitDiscoversGlobalAgents: the
// re-discovery path must resolve agent directories against the user home, not
// against the user config directory.
func TestConfigDiscoversGlobalAgents(t *testing.T) {
	origGetWd := getWd
	origGetUserConfig := getUserConfig
	origGetUserHome := getUserHome
	defer func() {
		getWd = origGetWd
		getUserConfig = origGetUserConfig
		getUserHome = origGetUserHome
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
			name:      "discovers global Gemini CLI",
			globalDir: ".gemini",
			want:      ".gemini/skills",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			homeDir := filepath.Join(root, "home")
			configDir := filepath.Join(root, "xdg_config")
			cwd := filepath.Join(root, "project")

			if err := os.MkdirAll(filepath.Join(homeDir, tt.globalDir), 0755); err != nil {
				t.Fatalf("failed to seed agent dir: %v", err)
			}
			if err := os.MkdirAll(cwd, 0755); err != nil {
				t.Fatalf("failed to seed cwd: %v", err)
			}

			cfgPath := filepath.Join(configDir, "skill-sync", "skill-sync.yaml")
			if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
				t.Fatalf("failed to seed config dir: %v", err)
			}
			if err := os.WriteFile(cfgPath, []byte("targets:\n  - stale\n"), 0644); err != nil {
				t.Fatalf("failed to seed config: %v", err)
			}

			getWd = func() (string, error) { return cwd, nil }
			getUserConfig = func() (string, error) { return configDir, nil }
			getUserHome = func() (string, error) { return homeDir, nil }

			cmd := NewConfigCmd()
			var out bytes.Buffer
			cmd.SetOut(&out)
			cmd.SetArgs([]string{})

			if err := cmd.Execute(); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			content, err := os.ReadFile(cfgPath)
			if err != nil {
				t.Fatalf("failed to read updated config: %v", err)
			}

			if !strings.Contains(string(content), tt.want) {
				t.Errorf("expected config to contain %q, got:\n%s", tt.want, string(content))
			}
		})
	}
}
