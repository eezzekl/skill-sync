package cli

import (
	"fmt"
	"os"
	"sort"

	"github.com/ezzek/skill-sync/internal/agent"
	"github.com/ezzek/skill-sync/internal/agent/config_resolver"
	"github.com/ezzek/skill-sync/internal/agent/discovery"
	"github.com/ezzek/skill-sync/internal/mesh"
	"github.com/ezzek/skill-sync/internal/models"
	internalsync "github.com/ezzek/skill-sync/internal/sync"
	"github.com/spf13/cobra"
)

func NewSyncCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:          "sync",
		Short:        "Synchronize skills across all configured target directories",
		SilenceUsage: true,
	}

	cmd.Flags().StringVarP(&configPath, "config", "c", "",
		"Path to the skill-sync configuration file")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		return runSync(c, configPath, nil)
	}

	return cmd
}

// ScanSkillInfos returns the sync state of every skill that needs action across configured targets.
// Skills already in sync everywhere are excluded. Conflicts are included with IsConflict=true.
func ScanSkillInfos(configPath string) ([]models.SkillSyncInfo, error) {
	cfgResolver := config_resolver.New()
	resolvedPath, err := cfgResolver.Resolve(configPath)
	if err != nil {
		return nil, fmt.Errorf("no configuration found; run 'skill-sync init' to create one")
	}

	f, err := os.Open(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open config: %w", err)
	}
	defer f.Close()

	cfg, err := agent.ParseConfig(f)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if home, err := os.UserHomeDir(); err == nil {
		cfg.ExpandTargets(home)
	}

	scanner := mesh.NewDefaultScanner()
	skillMap, err := scanner.Scan(cfg.Targets)
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	resolver := mesh.NewDefaultResolver()
	var infos []models.SkillSyncInfo

	for skillID, instances := range skillMap {
		winner, hasConflict, resolveErr := resolver.Resolve(instances)
		if resolveErr != nil {
			continue
		}
		if hasConflict {
			infos = append(infos, models.SkillSyncInfo{ID: skillID, IsConflict: true})
			continue
		}

		var outdatedTargets []string
		presentTargets := make(map[string]bool)
		for _, inst := range instances {
			presentTargets[inst.TargetDir] = true
			if inst.Hash != winner.Hash {
				outdatedTargets = append(outdatedTargets, inst.TargetDir)
			}
		}
		for _, target := range cfg.Targets {
			if !presentTargets[target] {
				outdatedTargets = append(outdatedTargets, target)
			}
		}
		if len(outdatedTargets) == 0 {
			continue
		}

		targetAgents := make([]string, len(outdatedTargets))
		for i, t := range outdatedTargets {
			targetAgents[i] = discovery.AgentNameFromPath(t)
		}
		infos = append(infos, models.SkillSyncInfo{
			ID:           skillID,
			SourceAgent:  discovery.AgentNameFromPath(winner.TargetDir),
			TargetAgents: targetAgents,
		})
	}

	sort.Slice(infos, func(i, j int) bool { return infos[i].ID < infos[j].ID })
	return infos, nil
}

// runSync synchronizes skills. If skillFilter is non-nil, only the listed skill IDs are synced.
func runSync(cmd *cobra.Command, configPath string, skillFilter []string) error {
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

	filterSet := make(map[string]bool, len(skillFilter))
	for _, id := range skillFilter {
		filterSet[id] = true
	}

	resolver := mesh.NewDefaultResolver()
	engine := internalsync.NewEngine()

	syncCount := 0
	conflictCount := 0

	for skillID, instances := range skillMap {
		if len(skillFilter) > 0 && !filterSet[skillID] {
			continue
		}

		winner, hasConflict, resolveErr := resolver.Resolve(instances)
		if resolveErr != nil {
			return fmt.Errorf("resolve failed for skill %s: %w", skillID, resolveErr)
		}

		if hasConflict {
			fmt.Fprintf(cmd.OutOrStdout(), "WARNING: conflict detected for skill %q — skipping\n", skillID)
			conflictCount++
			continue
		}

		var outdatedTargets []string
		presentTargets := make(map[string]bool)
		for _, inst := range instances {
			presentTargets[inst.TargetDir] = true
			if inst.Hash != winner.Hash {
				outdatedTargets = append(outdatedTargets, inst.TargetDir)
			}
		}
		for _, target := range cfg.Targets {
			if !presentTargets[target] {
				outdatedTargets = append(outdatedTargets, target)
			}
		}

		if len(outdatedTargets) == 0 {
			continue
		}

		if syncErr := engine.Sync(*winner, outdatedTargets); syncErr != nil {
			return fmt.Errorf("sync failed for skill %s: %w", skillID, syncErr)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s: v%g from %s → %d target(s) updated\n",
			skillID, winner.Metadata.Version, winner.TargetDir, len(outdatedTargets))
		syncCount++
	}

	fmt.Fprintf(cmd.OutOrStdout(), "sync complete: %d skill(s) updated, %d conflict(s) skipped\n", syncCount, conflictCount)
	return nil
}
