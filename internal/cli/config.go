package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ezzek/skill-sync/internal/agent/discovery"
	"github.com/ezzek/skill-sync/internal/writer"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func NewConfigCmd() *cobra.Command {
	var local bool
	var configPath string

	cmd := &cobra.Command{
		Use:          "config",
		Short:        "Update configuration by re-discovering targets",
		SilenceUsage: true,
	}

	cmd.Flags().BoolVar(&local, "local", false, "Update local project config (./skill-sync.yaml)")
	cmd.Flags().StringVarP(&configPath, "config", "c", "", "Explicit path to config file")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		return runConfig(c, local, configPath)
	}

	return cmd
}

func resolveConfigPath(local bool, explicitPath string) (string, error) {
	if explicitPath != "" {
		return explicitPath, nil
	}
	if local {
		cwd, err := getWd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
		return filepath.Join(cwd, "skill-sync.yaml"), nil
	}
	userDir, err := getUserConfig()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}
	return filepath.Join(userDir, "skill-sync", "skill-sync.yaml"), nil
}

func runConfig(cmd *cobra.Command, local bool, explicitPath string) error {
	configPath, err := resolveConfigPath(local, explicitPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("no configuration found at %s; run 'skill-sync init' to create one", configPath)
	}

	d := discovery.New()
	d.Getwd = getWd
	d.UserHomeDir = func() (string, error) { return getUserConfig() }

	discovered, err := d.Scan()
	if err != nil {
		return fmt.Errorf("failed to discover agents: %w", err)
	}

	targets := make([]string, len(discovered))
	for i, t := range discovered {
		targets[i] = filepath.ToSlash(filepath.Join(t, "skills"))
	}

	if err := updateConfigTargets(configPath, targets); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configuration updated at %s\n", configPath)
	return nil
}

func updateConfigTargets(filePath string, targets []string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return err
	}

	if len(root.Content) == 0 {
		return fmt.Errorf("empty yaml")
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		return fmt.Errorf("expected mapping node")
	}

	var targetsSeq *yaml.Node
	for i := 0; i < len(doc.Content); i += 2 {
		if doc.Content[i].Value == "targets" {
			targetsSeq = doc.Content[i+1]
			break
		}
	}

	if targetsSeq == nil {
		keyNode := &yaml.Node{Kind: yaml.ScalarNode, Value: "targets"}
		targetsSeq = &yaml.Node{Kind: yaml.SequenceNode}
		doc.Content = append(doc.Content, keyNode, targetsSeq)
	} else if targetsSeq.Kind != yaml.SequenceNode {
		return fmt.Errorf("expected targets to be a sequence")
	}

	targetsSeq.Content = make([]*yaml.Node, len(targets))
	for i, t := range targets {
		targetsSeq.Content[i] = &yaml.Node{Kind: yaml.ScalarNode, Value: t}
	}

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return err
	}
	enc.Close()

	return writer.AtomicWrite(filePath, buf.Bytes())
}
