package cli

import (
	"bytes"

	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/tui"
	"github.com/spf13/cobra"
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill-sync",
		Short: "skill-sync synchronizes AI agent skills across directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			callbacks := tui.Callbacks{
				ScanSkills: func() ([]models.SkillSyncInfo, error) {
					return ScanSkillInfos("")
				},
				RunSync: func(skillFilter []string) (string, error) {
					syncCmd := NewSyncCmd()
					var buf bytes.Buffer
					syncCmd.SetOut(&buf)
					syncCmd.SetErr(&buf)
					syncCmd.SilenceUsage = true
					syncCmd.SilenceErrors = true
					err := runSync(syncCmd, "", skillFilter)
					return buf.String(), err
				},
				RunVerify: func() (string, error) {
					verifyCmd := NewVerifyCmd()
					var buf bytes.Buffer
					verifyCmd.SetOut(&buf)
					verifyCmd.SetErr(&buf)
					verifyCmd.SilenceUsage = true
					verifyCmd.SilenceErrors = true
					err := verifyCmd.RunE(verifyCmd, nil)
					return buf.String(), err
				},
			}
			return tui.NewProgram(callbacks).Run()
		},
	}

	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewConfigCmd())
	cmd.AddCommand(NewSyncCmd())
	cmd.AddCommand(NewVerifyCmd())

	return cmd
}
