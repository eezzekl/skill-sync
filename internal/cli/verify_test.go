package cli_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ezzek/skill-sync/internal/cli"
)

func TestNewVerifyCmd(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T) string
		wantDrift  bool // true = expect exit code 1 (drift), false = clean
		wantErr    bool
	}{
		{
			name: "no drift when all skills are identical",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				content := "---\nversion: 1\n---\n# Stable Skill"
				writeSkillFile(t, toolA, "stable-skill", content)
				writeSkillFile(t, toolB, "stable-skill", content)

				return writeConfigFile(t, base, []string{toolA, toolB})
			},
			wantDrift: false,
			wantErr:   false,
		},
		{
			name: "detects drift when skill versions differ",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")
				toolB := filepath.Join(base, "toolB")

				writeSkillFile(t, toolA, "drifted-skill", "---\nversion: 2\n---\n# Drifted v2")
				writeSkillFile(t, toolB, "drifted-skill", "---\nversion: 1\n---\n# Drifted v1")

				return writeConfigFile(t, base, []string{toolA, toolB})
			},
			wantDrift: true,
			wantErr:   true, // verify exits with error (code 1) when drift detected
		},
		{
			name: "returns error for missing config file",
			setup: func(t *testing.T) string {
				return filepath.Join(t.TempDir(), "nonexistent.yaml")
			},
			wantDrift: false,
			wantErr:   true,
		},
		{
			name: "no drift with single target directory",
			setup: func(t *testing.T) string {
				base := t.TempDir()
				toolA := filepath.Join(base, "toolA")

				writeSkillFile(t, toolA, "solo-skill", "---\nversion: 1\n---\n# Solo Skill")
				return writeConfigFile(t, base, []string{toolA})
			},
			wantDrift: false,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfgPath := tt.setup(t)

			cmd := cli.NewVerifyCmd()
			cmd.SetArgs([]string{"--config", cfgPath})

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewVerifyCmd().Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyCmd_ReportsAllDriftedTargets(t *testing.T) {
	// Spec: when multiple targets are out of sync with the most current
	// version, verify must list EVERY drifted target — not just the first one.
	// The DRIFT line must also indicate the most current version and where it
	// lives so the user understands what the reference is.
	base := t.TempDir()
	toolA := filepath.Join(base, "toolA") // v1 (oldest)
	toolB := filepath.Join(base, "toolB") // v2 (middle — also drifted)
	toolC := filepath.Join(base, "toolC") // v3 (most current — reference)

	writeSkillFile(t, toolA, "git-expert", "---\nversion: 1\n---\n# Git Expert v1")
	writeSkillFile(t, toolB, "git-expert", "---\nversion: 2\n---\n# Git Expert v2")
	writeSkillFile(t, toolC, "git-expert", "---\nversion: 3\n---\n# Git Expert v3")

	cfgPath := writeConfigFile(t, base, []string{toolA, toolB, toolC})

	var out bytes.Buffer
	cmd := cli.NewVerifyCmd()
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--config", cfgPath})
	_ = cmd.Execute() // drift expected

	output := out.String()

	// Grouped format: one DRIFT line per skill listing all out-of-sync targets as short names
	driftCount := strings.Count(output, "DRIFT:")
	if driftCount != 1 {
		t.Errorf("expected 1 grouped DRIFT line for skill, got %d\noutput:\n%s", driftCount, output)
	}
	// AgentNameFromPath falls back to filepath.Base for unknown paths
	if !strings.Contains(output, "toolA") {
		t.Errorf("expected DRIFT line to include 'toolA'\noutput:\n%s", output)
	}
	if !strings.Contains(output, "toolB") {
		t.Errorf("expected DRIFT line to include 'toolB'\noutput:\n%s", output)
	}
	if !strings.Contains(output, "most current") {
		t.Errorf("expected DRIFT message to reference the most current version, got:\n%s", output)
	}
	if !strings.Contains(output, "v3") {
		t.Errorf("expected DRIFT message to include the most current version number (v3), got:\n%s", output)
	}
}

func TestVerifyCmd_DoesNotWriteFiles(t *testing.T) {
	// Spec: verify must NOT write files even when drift is detected
	base := t.TempDir()
	toolA := filepath.Join(base, "toolA")
	toolB := filepath.Join(base, "toolB")

	writeSkillFile(t, toolA, "test-skill", "---\nversion: 2\n---\n# New Version")
	writeSkillFile(t, toolB, "test-skill", "---\nversion: 1\n---\n# Old Version")

	cfgPath := writeConfigFile(t, base, []string{toolA, toolB})

	// Record original content of toolB before running verify
	origPath := filepath.Join(toolB, "skills", "test-skill", "SKILL.md")
	origContent, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("failed to read original file: %v", err)
	}

	cmd := cli.NewVerifyCmd()
	cmd.SetArgs([]string{"--config", cfgPath})
	_ = cmd.Execute() // error expected but we don't care about it here

	// Verify toolB was NOT modified
	afterContent, err := os.ReadFile(origPath)
	if err != nil {
		t.Fatalf("failed to read file after verify: %v", err)
	}
	if string(afterContent) != string(origContent) {
		t.Errorf("verify modified file content — it should be read-only\ngot:  %q\nwant: %q", string(afterContent), string(origContent))
	}

	// Verify NO .bak file was created (which would indicate a write occurred)
	bakPath := origPath + ".bak"
	if _, err := os.Stat(bakPath); err == nil {
		t.Errorf("verify created .bak file at %s — it should not write files", bakPath)
	}
}
