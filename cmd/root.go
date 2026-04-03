package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

var (
	flagBase string
	flagHead string
	flagPort int
)

var rootCmd = &cobra.Command{
	Use:   "umpire",
	Short: "Local code review tool",
	Long:  "Umpire provides a GitHub-like review UI in the browser for reviewing feature branch commits locally.",
	RunE:  runReview,
}

func init() {
	rootCmd.Flags().StringVar(&flagBase, "base", "main", "base branch to diff against")
	rootCmd.Flags().StringVar(&flagHead, "head", "", "head ref to review (default: current branch)")
	rootCmd.Flags().IntVar(&flagPort, "port", 0, "port to serve on (default: auto)")
}

func Execute() error {
	return rootCmd.Execute()
}

func runReview(cmd *cobra.Command, args []string) error {
	head := flagHead
	if head == "" {
		branch, err := currentBranch()
		if err != nil {
			return fmt.Errorf("detecting current branch: %w", err)
		}
		head = branch
	}

	fmt.Fprintf(os.Stderr, "Reviewing %s..%s\n", flagBase, head)
	return nil
}

func currentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
