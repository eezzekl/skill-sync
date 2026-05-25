package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ezzek/skill-sync/internal/agent/discovery"
	"github.com/ezzek/skill-sync/internal/writer"
	"github.com/spf13/cobra"
)

func NewInitCmd() *cobra.Command {
	var local bool
	var force bool

	cmd := &cobra.Command{
		Use:          "init",
		Short:        "Initialize a new skill-sync configuration",
		SilenceUsage: true,
	}

	cmd.Flags().BoolVar(&local, "local", false, "Create configuration in current directory")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing configuration")

	cmd.RunE = func(c *cobra.Command, args []string) error {
		return runInit(c, local, force)
	}

	return cmd
}

var (
	getWd                   = os.Getwd
	getUserConfig           = os.UserConfigDir
	stdinReader   io.Reader = os.Stdin
	isTTY                   = func() bool {
		st, err := os.Stdin.Stat()
		if err != nil {
			return false
		}
		return st.Mode()&os.ModeCharDevice != 0
	}
)

func runInit(cmd *cobra.Command, local, force bool) error {
	var configPath string

	if local {
		cwd, err := getWd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		configPath = filepath.Join(cwd, "skill-sync.yaml")
	} else {
		userDir, err := getUserConfig()
		if err != nil {
			return fmt.Errorf("failed to get user config directory: %w", err)
		}
		configDir := filepath.Join(userDir, "skill-sync")
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}
		configPath = filepath.Join(configDir, "skill-sync.yaml")
	}

	if !force {
		if _, err := os.Stat(configPath); err == nil {
			if !isTTY() {
				return fmt.Errorf("configuration already exists and stdin is not a terminal; re-run with --force to overwrite")
			}
			fmt.Fprintf(cmd.OutOrStdout(), "configuration already exists; reinitialize? [y/N]: ")
			reader := bufio.NewReader(stdinReader)
			line, _ := reader.ReadString('\n')
			if strings.ToLower(strings.TrimSpace(line)) != "y" {
				fmt.Fprintln(cmd.OutOrStdout(), "init aborted by user")
				return nil
			}
		}
	}

	d := discovery.New()
	d.Getwd = getWd
	d.UserHomeDir = func() (string, error) { return getUserConfig() }

	targets, err := d.Scan()
	if err != nil {
		return fmt.Errorf("failed to discover agents: %w", err)
	}

	if len(targets) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "no skills configured")
	}

	var buf bytes.Buffer
	fmt.Fprintln(&buf, "targets:")
	for _, t := range targets {
		skillsPath := filepath.Join(t, "skills")
		fmt.Fprintf(&buf, "  - %s\n", filepath.ToSlash(skillsPath))
	}

	if err := writer.AtomicWrite(configPath, buf.Bytes()); err != nil {
		return fmt.Errorf("failed to create config file atomically: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Initialized configuration at %s\n", configPath)
	return nil
}
