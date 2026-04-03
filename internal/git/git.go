package git

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Commit represents a single git commit.
type Commit struct {
	SHA     string `json:"sha"`
	Subject string `json:"subject"`
	Author  string `json:"author"`
	Date    string `json:"date"`
}

// ChangedFile represents a file changed between two refs.
type ChangedFile struct {
	Path   string `json:"path"`
	Status string `json:"status"` // A, M, D, R, etc.
}

// Repo provides git operations for a repository at a given path.
type Repo struct {
	Dir string
}

// NewRepo creates a Repo rooted at dir.
func NewRepo(dir string) *Repo {
	return &Repo{Dir: dir}
}

func (r *Repo) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.Dir
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), string(exitErr.Stderr))
		}
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}

// CurrentBranch returns the name of the current branch.
func (r *Repo) CurrentBranch() (string, error) {
	return r.run("rev-parse", "--abbrev-ref", "HEAD")
}

// MergeBase returns the merge base SHA between two refs.
func (r *Repo) MergeBase(a, b string) (string, error) {
	return r.run("merge-base", a, b)
}

// ResolveSHA resolves a ref to its full SHA.
func (r *Repo) ResolveSHA(ref string) (string, error) {
	return r.run("rev-parse", ref)
}

// CommitsBetween returns commits from base..head in reverse chronological order.
func (r *Repo) CommitsBetween(base, head string) ([]Commit, error) {
	out, err := r.run("log", "--format=%H\x1f%s\x1f%an\x1f%aI", base+".."+head)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var commits []Commit
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\x1f", 4)
		if len(parts) != 4 {
			continue
		}
		commits = append(commits, Commit{
			SHA:     parts[0],
			Subject: parts[1],
			Author:  parts[2],
			Date:    formatDate(parts[3]),
		})
	}
	return commits, nil
}

// DiffBetween returns the unified diff between two refs.
func (r *Repo) DiffBetween(base, head string) (string, error) {
	return r.run("diff", base+"..."+head)
}

// DiffForCommit returns the unified diff for a single commit.
func (r *Repo) DiffForCommit(sha string) (string, error) {
	return r.run("diff-tree", "-p", "--no-commit-id", sha)
}

// ChangedFiles returns the list of files changed between two refs.
func (r *Repo) ChangedFiles(base, head string) ([]ChangedFile, error) {
	out, err := r.run("diff", "--name-status", base+"..."+head)
	if err != nil {
		return nil, err
	}
	if out == "" {
		return nil, nil
	}

	var files []ChangedFile
	for _, line := range strings.Split(out, "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		files = append(files, ChangedFile{
			Status: parts[0],
			Path:   parts[1],
		})
	}
	return files, nil
}

func formatDate(isoDate string) string {
	t, err := time.Parse(time.RFC3339, isoDate)
	if err != nil {
		return isoDate
	}
	return t.Format("2006-01-02 15:04")
}
