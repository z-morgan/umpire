package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/zmorgan/umpire/internal/browser"
	"github.com/zmorgan/umpire/internal/feedback"
	gitpkg "github.com/zmorgan/umpire/internal/git"
	"github.com/zmorgan/umpire/internal/review"
	"github.com/zmorgan/umpire/internal/server"
)

var (
	flagBase string
	flagHead string
	flagPort int
)

func SetVersion(v string) {
	rootCmd.Version = v
}

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

	baseSHA, err := repo.ResolveSHA(flagBase)
	if err != nil {
		return fmt.Errorf("resolving base ref %q: %w", flagBase, err)
	}

	headSHA, err := repo.ResolveSHA(head)
	if err != nil {
		return fmt.Errorf("resolving head ref %q: %w", head, err)
	}

	mergeBase, err := repo.MergeBase(baseSHA, headSHA)
	if err != nil {
		return fmt.Errorf("finding merge base: %w", err)
	}

	commits, err := repo.CommitsBetween(mergeBase, headSHA)
	if err != nil {
		return fmt.Errorf("listing commits: %w", err)
	}

	files, err := repo.ChangedFiles(mergeBase, headSHA)
	if err != nil {
		return fmt.Errorf("listing changed files: %w", err)
	}

	store := review.NewStore(cwd)

	feedbackStore, err := feedback.NewStore()
	if err != nil {
		return fmt.Errorf("creating feedback store: %w", err)
	}

	shutdownCh := make(chan struct{}, 1)

	rc := &server.ReviewContext{
		Repo:          repo,
		BaseRef:       flagBase,
		HeadRef:       head,
		BaseSHA:       baseSHA,
		HeadSHA:       headSHA,
		MergeBase:     mergeBase,
		Store:         store,
		FeedbackStore: feedbackStore,
		ShutdownFn: func() {
			select {
			case shutdownCh <- struct{}{}:
			default:
			}
		},
	}

	srv, err := server.New(flagPort)
	if err != nil {
		return fmt.Errorf("starting server: %w", err)
	}

	server.RegisterAPI(srv.Mux(), rc)

	url := srv.URL()

	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintf(os.Stderr, "  umpire — local code review\n")
	fmt.Fprintf(os.Stderr, "  %s..%s\n", flagBase, head)
	fmt.Fprintf(os.Stderr, "  %d commits, %d files changed\n", len(commits), len(files))
	fmt.Fprintf(os.Stderr, "  %s\n", url)
	fmt.Fprintln(os.Stderr, "")

	suggestGitignore(cwd)

	browser.Open(url)

	// Graceful shutdown on interrupt or review submission
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
	case <-shutdownCh:
		fmt.Fprintln(os.Stderr, "\nShutting down...")
		// Brief delay so the browser gets the response
		time.Sleep(500 * time.Millisecond)
		return srv.Shutdown(context.Background())
	case err := <-errCh:
		return err
	}
}

func suggestGitignore(repoDir string) {
	path := filepath.Join(repoDir, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  hint: add .umpire/ to your .gitignore\n\n")
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == ".umpire/" || line == ".umpire" {
			return
		}
	}
	fmt.Fprintf(os.Stderr, "  hint: add .umpire/ to your .gitignore\n\n")
}
