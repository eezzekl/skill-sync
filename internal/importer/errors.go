package importer

import "errors"

// ErrSkillNotFound is returned when no skill matching the requested ID
// is found in any of the configured source directories.
var ErrSkillNotFound = errors.New("skill not found in any configured source")

// ErrConflict is returned when a skill exists in multiple sources with
// identical version and mtime but different content (true tie — no winner
// can be automatically determined).
var ErrConflict = errors.New("conflict: identical version and mtime in multiple sources")
