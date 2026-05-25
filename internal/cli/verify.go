package cli

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ezzek/skill-sync/internal/agent"
	"github.com/ezzek/skill-sync/internal/agent/config_resolver"
	"github.com/ezzek/skill-sync/internal/agent/discovery"
	"github.com/ezzek/skill-sync/internal/mesh"
	"github.com/spf13/cobra"
)

// NewVerifyCmd returns the `verify` subcommand which runs mesh resolution
// in read-only mode and exits with code 1 if any drift is detected.
// It NEVER writes files.
func NewVerifyCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:          "verify",
		Short:        "Check if all skills are in sync across configured directories (read-only)",
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "",
		"Path to the skill-sync configuration file")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		return runVerify(c, configPath)
	}

	return cmd
}

// ErrDriftDetected is returned by runVerify when one or more skills are out of sync.
var ErrDriftDetected = errors.New("drift detected: skills are not in sync across all targets")

type driftEntry struct {
	dirs      []string
	winnerVer float64
	winnerDir string
}

func runVerify(cmd *cobra.Command, configPath string) error {
	cfgResolver := config_resolver.New()
	resolvedPath, err := cfgResolver.Resolve(configPath)
	if err != nil {
		return fmt.Errorf("no configuration found; run 'skill-sync init' to create one")
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to open config %s: %w", configPath, err)
	}
	defer f.Close()

	cfg, err := agent.ParseConfig(f)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if home, err := os.UserHomeDir(); err == nil {
		cfg.ExpandTargets(home)
	}

	scanner := mesh.NewDefaultScanner()
	skillMap, err := scanner.Scan(cfg.Targets)
	if err != nil {
		return fmt.Errorf("scan failed: %w", err)
	}

	resolver := mesh.NewDefaultResolver()

	driftedSet := make(map[string]struct{})
	var driftedSkills []string
	driftBySkill := make(map[string]*driftEntry)
	missingBySkill := make(map[string][]string)

	markDrifted := func(skillID string) {
		if _, already := driftedSet[skillID]; !already {
			driftedSet[skillID] = struct{}{}
			driftedSkills = append(driftedSkills, skillID)
		}
	}

	for skillID, instances := range skillMap {
		winner, hasConflict, resolveErr := resolver.Resolve(instances)
		if resolveErr != nil {
			return fmt.Errorf("resolve failed for skill %s: %w", skillID, resolveErr)
		}

		if hasConflict {
			fmt.Fprintf(cmd.OutOrStdout(), "CONFLICT: skill %q has unresolvable conflict — manual resolution required\n", skillID)
			markDrifted(skillID)
			continue
		}

		for _, inst := range instances {
			if inst.Hash != winner.Hash {
				if driftBySkill[skillID] == nil {
					driftBySkill[skillID] = &driftEntry{
						winnerVer: winner.Metadata.Version,
						winnerDir: winner.TargetDir,
					}
				}
				driftBySkill[skillID].dirs = append(driftBySkill[skillID].dirs, inst.TargetDir)
				markDrifted(skillID)
			}
		}

		presentTargets := make(map[string]bool)
		for _, inst := range instances {
			presentTargets[inst.TargetDir] = true
		}
		for _, target := range cfg.Targets {
			if !presentTargets[target] {
				missingBySkill[skillID] = append(missingBySkill[skillID], target)
				markDrifted(skillID)
			}
		}
	}

	for skillID, entry := range driftBySkill {
		names := make([]string, len(entry.dirs))
		for i, d := range entry.dirs {
			names[i] = discovery.AgentNameFromPath(d)
		}
		fmt.Fprintf(cmd.OutOrStdout(),
			"DRIFT: skill %q — [%s] out of sync; most current: v%g in %s\n",
			skillID, strings.Join(names, ", "), entry.winnerVer, discovery.AgentNameFromPath(entry.winnerDir))
	}
	for skillID, dirs := range missingBySkill {
		names := make([]string, len(dirs))
		for i, d := range dirs {
			names[i] = discovery.AgentNameFromPath(d)
		}
		fmt.Fprintf(cmd.OutOrStdout(),
			"MISSING: skill %q — not found in [%s]\n",
			skillID, strings.Join(names, ", "))
	}

	if len(driftedSkills) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\nverify FAILED: %d skill(s) out of sync: %v\n", len(driftedSkills), driftedSkills)
		return ErrDriftDetected
	}

	fmt.Fprintf(cmd.OutOrStdout(), "verify OK: all skills are in sync across %d target(s)\n", len(cfg.Targets))
	return nil
}
