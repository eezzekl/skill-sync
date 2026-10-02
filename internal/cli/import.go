package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ezzek/skill-sync/internal/agent"
	"github.com/ezzek/skill-sync/internal/agent/config_resolver"
	"github.com/ezzek/skill-sync/internal/importer"
	"github.com/spf13/cobra"
)

// NewImportCmd returns the Cobra command for importing a single skill.
func NewImportCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:          "import <skill-name>",
		Short:        "Import a skill from a configured agent source into ./.agents/skills/",
		Args:         cobra.ExactArgs(1),
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := getWd()
			if err != nil {
				return fmt.Errorf("getwd: %w", err)
			}
			return runImport(cmd, configPath, cwd, args[0])
		},
	}
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to skill-sync.yaml")
	return cmd
}

func runImport(cmd *cobra.Command, configPath, cwd, skillID string) error {
	// 1. Resolve config.
	resolvedPath, err := config_resolver.New().Resolve(configPath)
	if err != nil {
		return fmt.Errorf("no skill-sync.yaml found; run 'skill-sync init' to create one")
	}
	f, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	cfg, err := agent.ParseConfig(f)
	if err != nil {
		return fmt.Errorf("parse config: %w", err)
	}
	if home, err := os.UserHomeDir(); err == nil {
		cfg.ExpandTargets(home)
	}

	// 2. Build importer.
	destRoot := filepath.Join(cwd, ".agents", "skills")
	imp := importer.New(cfg.Targets, destRoot)

	// 3. Find candidates.
	candidates, err := imp.FindSkill(skillID)
	if err != nil {
		return err
	}
	if len(candidates) == 0 {
		return fmt.Errorf("skill %q not found in any configured source", skillID)
	}

	// 4. Resolve winner.
	winner, _, err := importer.Resolve(candidates)
	if err != nil {
		if errors.Is(err, importer.ErrConflict) {
			return fmt.Errorf("conflict: %q has identical version and mtime in multiple sources; resolve manually", skillID)
		}
		return err
	}

	// 5. Copy.
	if err := imp.CopySkill(skillID, winner.SourceDir); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Imported %s from %s\n", skillID, winner.SourceDir)
	return nil
}