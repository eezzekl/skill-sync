package cli

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/ezzek/skill-sync/internal/importer"
	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/tui"
	"github.com/spf13/cobra"
)

func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skill-sync",
		Short:   "skill-sync synchronizes AI agent skills across directories",
		Version: version,
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
				FindSkillsForImport: func(sources []string) ([]importer.SkillCandidate, error) {
					cwd, err := getWd()
					if err != nil {
						return nil, err
					}
					destRoot := filepath.Join(cwd, ".agents", "skills")
					imp := importer.New(sources, destRoot)
					return imp.FindAllSkills()
				},
				RunImport: func(candidate importer.SkillCandidate) (string, error) {
					if candidate.IsConflict {
						return "", importer.ErrConflict
					}
					cwd, err := getWd()
					if err != nil {
						return "", err
					}
					destRoot := filepath.Join(cwd, ".agents", "skills")
					imp := importer.New(nil, destRoot)
					if err := imp.CopySkill(candidate.SkillID, candidate.SourceDir); err != nil {
						return "", err
					}
					return fmt.Sprintf("Imported %s from %s", candidate.SkillID, candidate.SourceDir), nil
				},
			}
			return tui.NewProgram(callbacks).Run()
		},
	}

	cmd.AddCommand(NewInitCmd())
	cmd.AddCommand(NewConfigCmd())
	cmd.AddCommand(NewSyncCmd())
	cmd.AddCommand(NewVerifyCmd())
	cmd.AddCommand(NewImportCmd())

	return cmd
}
