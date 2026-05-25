package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ezzek/skill-sync/internal/cli"
)

// helper: create a SKILL.md file with given content at <dir>/skills/<skillID>/SKILL.md
func writeSkillFile(t *testing.T, dir, skillID, content string) {
	t.Helper()
	skillDir := filepath.Join(dir, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}
}

// helper: write a skill-sync.yaml with two targets
func writeConfigFile(t *testing.T, dir string, targets []string) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("targets:\n")
	for _, tgt := range targets {
		sb.WriteString("  - " + tgt + "\n")
	}
	cfgPath := filepath.Join(dir, "skill-sync.yaml")
	if err := os.WriteFile(cfgPath, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return cfgPath
}

func TestSyncCmd_PrintsSkillDetailLine(t *testing.T) {
	base := t.TempDir()
	toolA := filepath.Join(base, "toolA")
	toolB := filepath.Join(base, "toolB")
	toolC := filepath.Join(base, "toolC")

	writeSkillFile(t, toolA, "git-expert", "---\nversion: 3\n---\n# Git Expert v3")
	writeSkillFile(t, toolB, "git-expert", "---\nversion: 1\n---\n# Git Expert v1")
	writeSkillFile(t, toolC, "git-expert", "---\nversion: 2\n---\n# Git Expert v2")

	cfgPath := writeConfigFile(t, base, []string{toolA, toolB, toolC})

	var out strings.Builder
	cmd := cli.NewSyncCmd()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"--config", cfgPath})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync returned unexpected error: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "git-expert") {
		t.Errorf("expected skill name in output, got:\n%s", output)
	}
	if !strings.Contains(output, "v3") {
		t.Errorf("expected winner version (v3) in output, got:\n%s", output)
	}
	if !strings.Contains(output, toolA) {
		t.Errorf("expected winner source dir in output, got:\n%s", output)
	}
	if !strings.Contains(output, "2 target(s) updated") {
		t.Errorf("expected updated count (2 target(s) updated) in output, got:\n%s", output)
	}
}

func TestNewSyncCmd(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) (cfgPath string)
		verify  func(t *testing.T, toolA, toolB string)
		wantErr bool
	}{
		{
			name: "propagates winner to outdated target",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				// toolA has version 2 (the winner)
				writeSkillFile(t, toolA, "my-skill", "---\nversion: 2\n---\n# My Skill v2")
				// toolB has version 1 (outdated)
				writeSkillFile(t, toolB, "my-skill", "---\nversion: 1\n---\n# My Skill v1")

				return writeConfigFile(t, base, []string{toolA, toolB})
			},
			verify: func(t *testing.T, toolA, toolB string) {
				// toolB should now have version 2 content
				got, err := os.ReadFile(filepath.Join(toolB, "skills", "my-skill", "SKILL.md"))
				if err != nil {
					t.Fatalf("failed to read synced file: %v", err)
				}
				if !strings.Contains(string(got), "# My Skill v2") {
					t.Errorf("expected synced content to contain '# My Skill v2', got: %s", string(got))
				}
			},
			wantErr: false,
		},
		{
			name: "no-op when all targets are in sync",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				content := "---\nversion: 1\n---\n# Identical Skill"
				writeSkillFile(t, toolA, "identical-skill", content)
				writeSkillFile(t, toolB, "identical-skill", content)

				return writeConfigFile(t, base, []string{toolA, toolB})
			},
			verify: func(t *testing.T, toolA, toolB string) {
				// Both targets remain unchanged - verify B still has identical content
				got, err := os.ReadFile(filepath.Join(toolB, "skills", "identical-skill", "SKILL.md"))
				if err != nil {
					t.Fatalf("failed to read file: %v", err)
				}
				if !strings.Contains(string(got), "# Identical Skill") {
					t.Errorf("expected unchanged content, got: %s", string(got))
				}
			},
			wantErr: false,
		},
		{
			name: "returns error for missing config file",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			verify:  func(t *testing.T, toolA, toolB string) {},
			wantErr: true,
		},
		{
			// W6: The presentTargets code path in sync.go:86-90 adds configured
			// targets that have NO SKILL.md files at all to the outdated list.
			// This test verifies that a completely empty target directory receives
			// the skill file after sync runs.
			name: "propagates skill to a target directory that is entirely empty",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				// toolA has the skill — toolB has NOTHING (not even the skill dir)
				writeSkillFile(t, toolA, "new-skill", "---\nversion: 1\n---\n# New Skill")
				// toolB is registered as a target but is completely empty:
				// do NOT call writeSkillFile for toolB — just create the directory.
				if err := os.MkdirAll(toolB, 0755); err != nil {
					t.Fatalf("failed to create empty toolB: %v", err)
				}

				return writeConfigFile(t, base, []string{toolA, toolB})
			},
			verify: func(t *testing.T, toolA, toolB string) {
				// toolB must now contain the skill file propagated from toolA
				destPath := filepath.Join(toolB, "skills", "new-skill", "SKILL.md")
				got, err := os.ReadFile(destPath)
				if err != nil {
					t.Fatalf("expected skill file to be created in empty target, but read failed: %v", err)
				}
				if !strings.Contains(string(got), "# New Skill") {
					t.Errorf("propagated content does not match: got %q, want content containing '# New Skill'", string(got))
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := tt.setup(t)

			// Derive toolA and toolB from config path for verify
			cfgDir := filepath.Dir(cfgPath)
			toolA := filepath.Join(cfgDir, "toolA")
			toolB := filepath.Join(cfgDir, "toolB")

			cmd := cli.NewSyncCmd()
			cmd.SetArgs([]string{"--config", cfgPath})

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewSyncCmd().Execute() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				tt.verify(t, toolA, toolB)
			}
		})
	}
}

func TestSyncCmd_TildeExpansion(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("USERPROFILE", homeDir)

	toolA := filepath.Join(homeDir, "toolA")
	toolB := filepath.Join(homeDir, "toolB")

	writeSkillFile(t, toolA, "tilde-skill", "---\nversion: 2\n---\n# Tilde Skill v2")
	writeSkillFile(t, toolB, "tilde-skill", "---\nversion: 1\n---\n# Tilde Skill v1")

	targets := []string{"~/toolA", "~/toolB"}
	cfgPath := writeConfigFile(t, homeDir, targets)

	cmd := cli.NewSyncCmd()
	cmd.SetArgs([]string{"--config", cfgPath})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("sync returned unexpected error: %v", err)
	}

	destPath := filepath.Join(toolB, "skills", "tilde-skill", "SKILL.md")
	got, err := os.ReadFile(destPath)
	if err != nil {
		t.Fatalf("failed to read synced file: %v", err)
	}
	if !strings.Contains(string(got), "# Tilde Skill v2") {
		t.Errorf("expected synced content to contain '# Tilde Skill v2', got: %s", string(got))
	}
}
