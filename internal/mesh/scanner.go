package mesh

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ezzek/skill-sync/internal/models"
	"gopkg.in/yaml.v3"
)

type Scanner interface {
	Scan(targetDirs []string) (map[string][]models.SkillInstance, error)
}

type defaultScanner struct{}

func NewDefaultScanner() Scanner {
	return &defaultScanner{}
}

func (s *defaultScanner) Scan(targetDirs []string) (map[string][]models.SkillInstance, error) {
	result := make(map[string][]models.SkillInstance)

	for _, targetDir := range targetDirs {
		skillsDir := targetDir
		if filepath.Base(targetDir) != "skills" {
			skillsDir = filepath.Join(targetDir, "skills")
		}

		err := filepath.WalkDir(skillsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) && path == skillsDir {
					return filepath.SkipDir
				}
				return err
			}

			// We are only looking for SKILL.md files
			if !d.IsDir() && d.Name() == "SKILL.md" {
				// Get the skill ID from the parent directory
				skillID := filepath.Base(filepath.Dir(path))

				content, err := os.ReadFile(path)
				if err != nil {
					return err
				}

				// Compute hash
				hash := sha256.Sum256(content)
				hashStr := hex.EncodeToString(hash[:])

				// Parse frontmatter
				metadata := parseFrontmatter(content)

				// Get mtime
				info, err := d.Info()
				if err != nil {
					return err
				}
				mtime := info.ModTime()

				instance := models.SkillInstance{
					ID:        skillID,
					Path:      path,
					TargetDir: targetDir,
					Hash:      hashStr,
					Mtime:     mtime,
					Metadata:  metadata,
				}

				result[skillID] = append(result[skillID], instance)
			}
			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to scan %s: %w", skillsDir, err)
		}
	}

	return result, nil
}

func parseFrontmatter(content []byte) models.SkillMetadata {
	str := string(content)

	// Check if the file starts with "---"
	if !strings.HasPrefix(str, "---\n") && !strings.HasPrefix(str, "---\r\n") {
		return models.SkillMetadata{}
	}

	// Find the end of the frontmatter
	endIndex := strings.Index(str[3:], "\n---")
	if endIndex == -1 {
		return models.SkillMetadata{} // Invalid frontmatter format
	}

	// Extract the YAML part
	yamlStr := str[3 : endIndex+3] // +3 to account for the initial "---"

	var metadata models.SkillMetadata
	_ = yaml.Unmarshal([]byte(yamlStr), &metadata)

	return metadata
}
