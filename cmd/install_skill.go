package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/zmorgan/umpire/skills"
)

var (
	flagProject bool
	flagForce   bool
)

var installSkillCmd = &cobra.Command{
	Use:   "install-skill",
	Short: "Install the /umpire skill for Claude Code",
	Long: "Write the /umpire skill to disk so a Claude Code session can start a review and act on the result.\n\n" +
		"Installs to ~/.claude/skills/umpire/SKILL.md by default. Umpire works in any git\n" +
		"repository, so a user-level install covers every repo at once; --project scopes it\n" +
		"to the current one instead.",
	Args: cobra.NoArgs,
	RunE: runInstallSkill,
}

func init() {
	installSkillCmd.Flags().BoolVar(&flagProject, "project", false, "install to ./.claude/skills instead of ~/.claude/skills")
	installSkillCmd.Flags().BoolVar(&flagForce, "force", false, "overwrite an existing SKILL.md")
	rootCmd.AddCommand(installSkillCmd)
}

func runInstallSkill(cmd *cobra.Command, args []string) error {
	path, err := skillPath(flagProject)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); err == nil && !flagForce {
		return fmt.Errorf("%s already exists; pass --force to overwrite it", path)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating skill directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(skills.UmpireSKILL), 0o644); err != nil {
		return fmt.Errorf("writing skill: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "umpire: skill installed to %s\n", path)
	fmt.Fprintf(cmd.OutOrStdout(), "Run /umpire in a Claude Code session to start a review.\n")
	return nil
}

// skillPath resolves where SKILL.md belongs. Project installs are relative to
// the working directory; user installs go under the home directory, which is
// the default because umpire runs in any git repository and a project-scoped
// install would have to be repeated in each one.
func skillPath(project bool) (string, error) {
	var root string
	var err error
	if project {
		root, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting working directory: %w", err)
		}
	} else {
		root, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("finding home directory: %w", err)
		}
	}
	return filepath.Join(root, ".claude", "skills", "umpire", "SKILL.md"), nil
}
