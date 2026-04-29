package agent_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/ezzek/skill-sync/internal/agent"
)

func TestParseConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected *agent.Config
		wantErr  bool
	}{
		{
			name: "Valid config load",
			input: `
include:
  - "**/*.md"
exclude:
  - "node_modules/**"
targets:
  - "~/.claude/skills"
  - "~/.config/opencode/skills"
`,
			expected: &agent.Config{
				IncludeGlobs: []string{"**/*.md"},
				ExcludeGlobs: []string{"node_modules/**"},
				Targets: []string{
					"~/.claude/skills",
					"~/.config/opencode/skills",
				},
			},
			wantErr: false,
		},
		{
			name:    "Invalid YAML",
			input:   `include: - "broken`,
			wantErr: true,
		},
		{
			name: "Empty config",
			input: ``,
			expected: &agent.Config{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			cfg, err := agent.ParseConfig(reader)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(cfg, tt.expected) {
				t.Errorf("ParseConfig() = %v, want %v", cfg, tt.expected)
			}
		})
	}
}

func TestResolvePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		home     string
		expected string
	}{
		{
			name:     "Expands tilde",
			input:    "~/.claude/skills",
			home:     "/home/user",
			expected: "/home/user/.claude/skills",
		},
		{
			name:     "No tilde",
			input:    "/var/log/skills",
			home:     "/home/user",
			expected: "/var/log/skills",
		},
		{
			name:     "Empty path",
			input:    "",
			home:     "/home/user",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := agent.ResolvePath(tt.input, tt.home)
			if got != tt.expected {
				t.Errorf("ResolvePath() = %v, want %v", got, tt.expected)
			}
		})
	}
}
