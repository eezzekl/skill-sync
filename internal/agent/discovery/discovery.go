package discovery

import (
	"os"
	"path/filepath"
)

// AgentNameFromPath returns a short human-readable agent name for a given path.
// It strips a trailing "skills" component before matching against known agent dirs.
// Falls back to the last path component if no match is found.
func AgentNameFromPath(path string) string {
	base := path
	if filepath.Base(base) == "skills" {
		base = filepath.Dir(base)
	}

	homeDir, _ := os.UserHomeDir()
	cwd, _ := os.Getwd()

	for _, agent := range SupportedAgents {
		if homeDir != "" {
			if filepath.Clean(filepath.Join(homeDir, agent.GlobalDir)) == filepath.Clean(base) {
				return agent.Name
			}
		}
		if cwd != "" && agent.LocalDir != "" {
			if filepath.Clean(filepath.Join(cwd, agent.LocalDir)) == filepath.Clean(base) {
				return agent.Name + " (local)"
			}
		}
	}
	return filepath.Base(path)
}

type AgentConfig struct {
	Name      string
	LocalDir  string // relative to CWD
	GlobalDir string // relative to User Home Dir
}

var SupportedAgents = []AgentConfig{
	{"OpenCode", ".opencode", filepath.Join(".config", "opencode")},
	{"Claude Code", ".claude", ".claude"},
	{"Cursor", ".cursor", ".cursor"},
	{"Gemini CLI", ".gemini", ".gemini"},
	{"Antigravity", ".antigravity", filepath.Join(".gemini", "antigravity")},
	{"Codex", ".codex", ".codex"},
	{"Windsurf", filepath.Join(".codeium", "windsurf"), filepath.Join(".codeium", "windsurf")},
	{"GitHub Copilot", ".copilot", ".copilot"},
	// Pi stores user-authored skills under ~/.pi/agent/skills/.
	// Package-installed skills (npm/node_modules) are excluded by design — they are managed by npm, not skill-sync.
	{"Pi", "", filepath.Join(".pi", "agent")},
}

type Discovery struct {
	Getwd       func() (string, error)
	UserHomeDir func() (string, error)
}

func New() *Discovery {
	return &Discovery{
		Getwd:       os.Getwd,
		UserHomeDir: os.UserHomeDir,
	}
}

// Scan returns a list of active directories (existing in filesystem).
func (d *Discovery) Scan() ([]string, error) {
	activeDirs := []string{}

	cwd, cwdErr := d.Getwd()
	homeDir, homeErr := d.UserHomeDir()

	for _, agent := range SupportedAgents {
		if cwdErr == nil && agent.LocalDir != "" {
			localPath := filepath.Join(cwd, agent.LocalDir)
			if stat, err := os.Stat(localPath); err == nil && stat.IsDir() {
				activeDirs = append(activeDirs, localPath)
			}
		}

		if homeErr == nil && agent.GlobalDir != "" {
			globalPath := filepath.Join(homeDir, agent.GlobalDir)
			if stat, err := os.Stat(globalPath); err == nil && stat.IsDir() {
				activeDirs = append(activeDirs, globalPath)
			}
		}
	}

	return activeDirs, nil
}
