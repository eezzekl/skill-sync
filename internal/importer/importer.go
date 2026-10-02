package importer

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ezzek/skill-sync/internal/mesh"
	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/writer"
	"gopkg.in/yaml.v3"
)

// SkillCandidate is a skill directory found during source discovery.
type SkillCandidate struct {
	SkillID    string               // directory name, e.g. "git-expert"
	SourceDir  string               // absolute path to the skill directory
	AgentDir   string               // absolute path to the parent target
	Metadata   models.SkillMetadata // parsed from SKILL.md frontmatter
	Mtime      time.Time            // mtime of SKILL.md
	Hash       string               // DirHash of SourceDir
	IsConflict bool                 // true if multiple sources are a true tie
}

// Importer drives import operations for a fixed set of source targets
// and a fixed destination root.
type Importer struct {
	Targets  []string // tilde-expanded absolute paths
	DestRoot string   // absolute path to ./.agents/skills/
}

// New returns an Importer.
// targets must be tilde-expanded absolute paths.
// destRoot is typically filepath.Join(cwd, ".agents", "skills").
func New(targets []string, destRoot string) *Importer {
	return &Importer{Targets: targets, DestRoot: destRoot}
}

// FindSkill walks each target's skills/ subdirectory and returns all
// candidates whose directory name matches skillID (case-sensitive).
// Returns an empty slice (not an error) when no match is found.
func (imp *Importer) FindSkill(skillID string) ([]SkillCandidate, error) {
	var candidates []SkillCandidate
	for _, target := range imp.Targets {
		skillDir := filepath.Join(target, "skills", skillID)
		mdPath := filepath.Join(skillDir, "SKILL.md")

		info, err := os.Stat(mdPath)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("stat %s: %w", mdPath, err)
		}

		meta, err := readMetadata(mdPath)
		if err != nil {
			meta = models.SkillMetadata{} // tolerate parse errors
		}

		hash, err := DirHash(skillDir)
		if err != nil {
			return nil, fmt.Errorf("hash %s: %w", skillDir, err)
		}

		candidates = append(candidates, SkillCandidate{
			SkillID:   skillID,
			SourceDir: skillDir,
			AgentDir:  target,
			Metadata:  meta,
			Mtime:     info.ModTime(),
			Hash:      hash,
		})
	}
	return candidates, nil
}

// FindAllSkills walks each target's skills/ subdirectory, groups candidates
// by SkillID, and resolves the winner for each unique skill ID. If a skill
// has a true tie conflict, it returns a candidate with IsConflict=true.
// Results are sorted by SkillID.
func (imp *Importer) FindAllSkills() ([]SkillCandidate, error) {
	candidatesByID := make(map[string][]SkillCandidate)

	for _, target := range imp.Targets {
		skillsDir := filepath.Join(target, "skills")
		entries, err := os.ReadDir(skillsDir)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", skillsDir, err)
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			skillID := e.Name()
			skillDir := filepath.Join(skillsDir, skillID)
			mdPath := filepath.Join(skillDir, "SKILL.md")

			info, err := os.Stat(mdPath)
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			if err != nil {
				continue
			}

			meta, _ := readMetadata(mdPath)
			hash, err := DirHash(skillDir)
			if err != nil {
				return nil, fmt.Errorf("hash %s: %w", skillDir, err)
			}
			candidatesByID[skillID] = append(candidatesByID[skillID], SkillCandidate{
				SkillID:   skillID,
				SourceDir: skillDir,
				AgentDir:  target,
				Metadata:  meta,
				Mtime:     info.ModTime(),
				Hash:      hash,
			})
		}
	}

	var results []SkillCandidate
	for _, group := range candidatesByID {
		winner, conflict, err := Resolve(group)
		if conflict {
			c := group[0]
			c.IsConflict = true
			results = append(results, c)
		} else if err == nil && winner != nil {
			results = append(results, *winner)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].SkillID < results[j].SkillID
	})

	return results, nil
}

// Resolve converts candidates to SkillInstances and delegates winner
// selection to mesh.NewDefaultResolver(). Returns the winning candidate,
// a conflict flag, and any error.
func Resolve(candidates []SkillCandidate) (*SkillCandidate, bool, error) {
	if len(candidates) == 0 {
		return nil, false, ErrSkillNotFound
	}
	if len(candidates) == 1 {
		c := candidates[0]
		return &c, false, nil
	}

	instances := make([]models.SkillInstance, len(candidates))
	for i, c := range candidates {
		instances[i] = models.SkillInstance{
			ID:        c.SkillID,
			Path:      filepath.Join(c.SourceDir, "SKILL.md"),
			TargetDir: c.AgentDir,
			Hash:      c.Hash,
			Mtime:     c.Mtime,
			Metadata:  c.Metadata,
		}
	}

	winner, conflict, err := mesh.NewDefaultResolver().Resolve(instances)
	if err != nil {
		return nil, false, err
	}
	if conflict {
		return nil, true, ErrConflict
	}

	// Re-map winner back to its SkillCandidate by matching AgentDir.
	for i := range candidates {
		if candidates[i].AgentDir == winner.TargetDir {
			c := candidates[i]
			return &c, false, nil
		}
	}
	return nil, false, fmt.Errorf("importer: resolve winner not found in candidates (internal error)")
}

// CopySkill copies the full skill directory from src into
// filepath.Join(destRoot, skillID) using a transactional strategy:
//  1. Write all files to <skillID>.tmp/ (each file via writer.AtomicWrite).
//  2. On failure: remove .tmp, return error — existing dir untouched.
//  3. On success: remove stale .bak, rename existing → .bak, rename .tmp → final.
func (imp *Importer) CopySkill(skillID, src string) error {
	if err := os.MkdirAll(imp.DestRoot, 0o755); err != nil {
		return fmt.Errorf("create dest root: %w", err)
	}

	destFinal := filepath.Join(imp.DestRoot, skillID)
	destTmp := filepath.Join(imp.DestRoot, skillID+".tmp")
	destBak := filepath.Join(imp.DestRoot, skillID+".bak")

	// Ensure a clean tmp before starting.
	_ = os.RemoveAll(destTmp)

	// Copy all files into .tmp/
	copyErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // skip symlinks silently
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		tmpDest := filepath.Join(destTmp, rel)
		if d.IsDir() {
			return os.MkdirAll(tmpDest, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", rel, err)
		}
		if err := os.MkdirAll(filepath.Dir(tmpDest), 0o755); err != nil {
			return err
		}
		return writer.AtomicWrite(tmpDest, data)
	})
	if copyErr != nil {
		_ = os.RemoveAll(destTmp) // cleanup staging dir on failure
		return fmt.Errorf("copy skill %s: %w", skillID, copyErr)
	}

	// Commit: rotate backup, rename tmp → final.
	_ = os.RemoveAll(destBak)
	if _, err := os.Stat(destFinal); err == nil {
		if err := os.Rename(destFinal, destBak); err != nil {
			_ = os.RemoveAll(destTmp)
			return fmt.Errorf("backup existing skill: %w", err)
		}
	}
	if err := os.Rename(destTmp, destFinal); err != nil {
		// Attempt to restore backup.
		if _, statErr := os.Stat(destBak); statErr == nil {
			_ = os.Rename(destBak, destFinal)
		}
		return fmt.Errorf("commit skill %s: %w", skillID, err)
	}
	return nil
}

// readMetadata parses the YAML frontmatter from a SKILL.md file.
// Returns zero-value SkillMetadata on any parse error.
func readMetadata(mdPath string) (models.SkillMetadata, error) {
	data, err := os.ReadFile(mdPath)
	if err != nil {
		return models.SkillMetadata{}, err
	}
	s := string(data)
	// Require a proper frontmatter fence (Unix or Windows line endings).
	if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
		return models.SkillMetadata{}, nil
	}
	// Find the closing fence.
	endIndex := strings.Index(s[3:], "\n---")
	if endIndex == -1 {
		return models.SkillMetadata{}, nil
	}
	yamlStr := s[3 : endIndex+3]
	var meta models.SkillMetadata
	_ = yaml.Unmarshal([]byte(yamlStr), &meta)
	return meta, nil
}
