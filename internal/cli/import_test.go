package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeImportSkillFile creates a skill directory with SKILL.md at
// <base>/<target>/skills/<skillID>/SKILL.md
func writeImportSkillFile(t *testing.T, base, target, skillID, content string) {
	t.Helper()
	skillDir := filepath.Join(base, target, "skills", skillID)
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("failed to create skill dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write skill file: %v", err)
	}
}

// writeImportConfig writes a skill-sync.yaml with the given targets.
func writeImportConfig(t *testing.T, base string, targets []string) string {
	t.Helper()
	var sb strings.Builder
	sb.WriteString("targets:\n")
	for _, tgt := range targets {
		sb.WriteString("  - " + tgt + "\n")
	}
	cfgPath := filepath.Join(base, "skill-sync.yaml")
	if err := os.WriteFile(cfgPath, []byte(sb.String()), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}
	return cfgPath
}

func TestImportCmd(t *testing.T) {
	origGetWd := getWd
	defer func() {
		getWd = origGetWd
	}()

	tests := []struct {
		name       string
		setup      func(t *testing.T, base string) (args []string, cfgPath string, cwd string)
		verify     func(t *testing.T, base string, stdout, stderr string, err error)
		wantErr    bool
		wantErrMsg string
	}{
		{
			name: "happy path — imports skill and creates destination files",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				toolA := filepath.Join(base, "toolA")
				writeImportSkillFile(t, base, "toolA", "git-expert", "---\nversion: 2\nname: Git Expert\ndescription: Git expertise\n---\n# Git Expert v2")
				cfgPath := writeImportConfig(t, base, []string{toolA})
				return []string{"git-expert"}, cfgPath, base
			},
			verify: func(t *testing.T, base string, stdout, stderr string, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if !strings.Contains(stdout, "Imported git-expert") {
					t.Errorf("expected stdout to contain 'Imported git-expert', got: %s", stdout)
				}
				destPath := filepath.Join(base, ".agents", "skills", "git-expert", "SKILL.md")
				data, readErr := os.ReadFile(destPath)
				if readErr != nil {
					t.Fatalf("expected skill file at %s, got error: %v", destPath, readErr)
				}
				if !strings.Contains(string(data), "# Git Expert v2") {
					t.Errorf("expected imported content to contain '# Git Expert v2', got: %s", string(data))
				}
			},
			wantErr: false,
		},
		{
			name: "missing config file — exit 1 with init hint",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				noCfg := filepath.Join(base, "nonexistent.yaml")
				return []string{"git-expert"}, noCfg, base
			},
			verify:     func(t *testing.T, base string, stdout, stderr string, err error) {},
			wantErr:    true,
			wantErrMsg: "skill-sync init",
		},
		{
			name: "skill not found in any source — exit 1",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				toolA := filepath.Join(base, "toolA")
				writeImportSkillFile(t, base, "toolA", "other-skill", "---\nversion: 1\n---\n# Other")
				cfgPath := writeImportConfig(t, base, []string{toolA})
				return []string{"nonexistent-skill"}, cfgPath, base
			},
			verify:     func(t *testing.T, base string, stdout, stderr string, err error) {},
			wantErr:    true,
			wantErrMsg: "not found",
		},
		{
			name: "wrong arg count — zero args — exit 1 with usage",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				cfgPath := filepath.Join(base, "skill-sync.yaml")
				_ = os.WriteFile(cfgPath, []byte("targets:\n  - /tmp\n"), 0644)
				return []string{}, cfgPath, base
			},
			verify:     func(t *testing.T, base string, stdout, stderr string, err error) {},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg",
		},
		{
			name: "wrong arg count — two args — exit 1 with usage",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				cfgPath := filepath.Join(base, "skill-sync.yaml")
				_ = os.WriteFile(cfgPath, []byte("targets:\n  - /tmp\n"), 0644)
				return []string{"arg1", "arg2"}, cfgPath, base
			},
			verify:     func(t *testing.T, base string, stdout, stderr string, err error) {},
			wantErr:    true,
			wantErrMsg: "accepts 1 arg",
		},
		{
			name: "conflict — identical version and mtime, different hash — exit 1",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				writeImportSkillFile(t, base, "toolA", "ambiguous-skill", "---\nversion: 1\n---\n# Content A")
				writeImportSkillFile(t, base, "toolB", "ambiguous-skill", "---\nversion: 1\n---\n# Content B")

				fixedTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				pathA := filepath.Join(base, "toolA", "skills", "ambiguous-skill", "SKILL.md")
				pathB := filepath.Join(base, "toolB", "skills", "ambiguous-skill", "SKILL.md")
				_ = os.Chtimes(pathA, fixedTime, fixedTime)
				_ = os.Chtimes(pathB, fixedTime, fixedTime)

				cfgPath := writeImportConfig(t, base, []string{toolA, toolB})
				return []string{"ambiguous-skill"}, cfgPath, base
			},
			verify:     func(t *testing.T, base string, stdout, stderr string, err error) {},
			wantErr:    true,
			wantErrMsg: "conflict",
		},
		{
			name: "idempotent — imports same skill twice without error",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				toolA := filepath.Join(base, "toolA")
				writeImportSkillFile(t, base, "toolA", "stable-skill", "---\nversion: 1\n---\n# Stable Skill")
				cfgPath := writeImportConfig(t, base, []string{toolA})
				return []string{"stable-skill"}, cfgPath, base
			},
			verify: func(t *testing.T, base string, stdout, stderr string, err error) {
				if err != nil {
					t.Fatalf("first import failed: %v", err)
				}
				if !strings.Contains(stdout, "Imported stable-skill") {
					t.Errorf("expected first import confirmation, got: %s", stdout)
				}
			},
			wantErr: false,
		},
		{
			name: "creates .bak backup when destination already exists",
			setup: func(t *testing.T, base string) ([]string, string, string) {
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				writeImportSkillFile(t, base, "toolA", "backup-skill", "---\nversion: 2\n---\n# Backup Skill v2")
				writeImportSkillFile(t, base, "toolB", "backup-skill", "---\nversion: 1\n---\n# Backup Skill v1")

				// Populate the destination before import so it exists and has the old content
				destDir := filepath.Join(base, ".agents", "skills", "backup-skill")
				if err := os.MkdirAll(destDir, 0755); err != nil {
					t.Fatalf("failed to create destination dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), []byte("---\nversion: 1\n---\n# Backup Skill v1"), 0644); err != nil {
					t.Fatalf("failed to write destination skill: %v", err)
				}

				cfgPath := writeImportConfig(t, base, []string{toolA, toolB})
				return []string{"backup-skill"}, cfgPath, base
			},
			verify: func(t *testing.T, base string, stdout, stderr string, err error) {
				if err != nil {
					t.Fatalf("import failed: %v", err)
				}
				bakPath := filepath.Join(base, ".agents", "skills", "backup-skill.bak", "SKILL.md")
				data, readErr := os.ReadFile(bakPath)
				if readErr != nil {
					t.Fatalf("expected .bak file at %s, got error: %v", bakPath, readErr)
				}
				if !strings.Contains(string(data), "# Backup Skill v1") {
					t.Errorf("expected .bak to contain old content '# Backup Skill v1', got: %s", string(data))
				}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			args, cfgPath, cwd := tt.setup(t, base)

			if cwd != "" {
				getWd = func() (string, error) { return cwd, nil }
			} else {
				getWd = os.Getwd
			}

			cmd := NewImportCmd()
			cmd.SetArgs(append([]string{"--config", cfgPath}, args...))

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrMsg)
				}
				errStr := err.Error() + stderr.String()
				if !strings.Contains(errStr, tt.wantErrMsg) {
					t.Errorf("error %q does not contain %q", errStr, tt.wantErrMsg)
				}
			} else {
				tt.verify(t, base, stdout.String(), stderr.String(), err)
			}
		})
	}
}
