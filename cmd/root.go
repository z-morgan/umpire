package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"github.com/zmorgan/umpire/internal/browser"
	gitpkg "github.com/zmorgan/umpire/internal/git"
	"github.com/zmorgan/umpire/internal/server"
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
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	repo := gitpkg.NewRepo(cwd)

	head := flagHead
	if head == "" {
		branch, err := repo.CurrentBranch()
		if err != nil {
			return fmt.Errorf("detecting current branch: %w", err)
		}
		head = branch
	}

	fmt.Fprintf(os.Stderr, "Reviewing %s..%s\n", flagBase, head)

	srv, err := server.New(flagPort)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	url := srv.URL()
	fmt.Fprintf(os.Stderr, "Serving at %s\n", url)
	browser.Open(url)

	// Graceful shutdown on interrupt
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Serve()
	}()

	select {
	case <-ctx.Done():
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}
