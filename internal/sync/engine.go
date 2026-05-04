package sync

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/writer"
)

type Engine interface {
	Sync(winner models.SkillInstance, targetDirs []string) error
}

type engine struct{}

func NewEngine() Engine {
	return &engine{}
}

func (e *engine) Sync(winner models.SkillInstance, targetDirs []string) error {
	// Read winner file
	winnerFile, err := os.Open(winner.Path)
	if err != nil {
		return fmt.Errorf("failed to open winner file: %w", err)
	}
	defer winnerFile.Close()

	content, err := io.ReadAll(winnerFile)
	if err != nil {
		return fmt.Errorf("failed to read winner file: %w", err)
	}

	validTargets := 0
	for _, targetDir := range targetDirs {
		baseDir := filepath.Dir(targetDir)
		if stat, err := os.Stat(baseDir); err != nil || !stat.IsDir() {
			continue
		}

		if err := os.MkdirAll(targetDir, 0755); err != nil {
			return fmt.Errorf("failed to create agent dir: %w", err)
		}

		skillsDir := filepath.Join(targetDir, "skills")
		if err := os.MkdirAll(skillsDir, 0755); err != nil {
			return fmt.Errorf("failed to create skills dir: %w", err)
		}

		destDir := filepath.Join(skillsDir, winner.ID)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create skill dir: %w", err)
		}

		destPath := filepath.Join(destDir, "SKILL.md")

		if err := writer.AtomicWrite(destPath, content); err != nil {
			return err // AtomicWrite handles backup, temp file, sync, and rename
		}
		validTargets++
	}

	if validTargets == 0 {
		return fmt.Errorf("no target directories found")
	}

	return nil
}
