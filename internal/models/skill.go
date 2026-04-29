package models

import "time"

// SkillMetadata represents the parsed YAML frontmatter.
// Version is a float64 so decimal tags like "1.0", "1.1", "1.2" round-trip
// cleanly; plain integers ("version: 2") still parse as 2.0 and compare
// correctly against decimals.
type SkillMetadata struct {
	Version float64 `yaml:"version"`
	Name    string  `yaml:"name"`
}

// SkillInstance represents a single skill file found in a target directory.
type SkillInstance struct {
	ID        string
	Path      string
	TargetDir string
	Hash      string
	Mtime     time.Time
	Metadata  SkillMetadata
}

// SkillSyncInfo describes a skill and its pending sync state across targets.
// IsConflict marks skills where multiple versions compete and no winner can be chosen.
type SkillSyncInfo struct {
	ID           string
	SourceAgent  string   // winning instance's agent short name; empty when IsConflict
	TargetAgents []string // agents where this skill will be propagated; empty when IsConflict
	IsConflict   bool
}
