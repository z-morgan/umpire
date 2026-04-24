package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestRepo creates a temp git repo with a main branch and a feature branch
// containing two commits ahead of main.
func setupTestRepo(t *testing.T) (string, func()) {
	t.Helper()
	dir := t.TempDir()

	commands := [][]string{
		{"git", "init", "--initial-branch=main"},
		{"git", "config", "user.email", "test@test.com"},
		{"git", "config", "user.name", "Test"},
	}
	for _, args := range commands {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("setup %v: %s\n%s", args, err, out)
		}
	}

	// Create initial file on main
	writeFile(t, dir, "README.md", "# Test\n")
	gitAdd(t, dir, ".")
	gitCommit(t, dir, "Initial commit")

	// Create feature branch with two commits
	gitCheckout(t, dir, "-b", "feature")
	writeFile(t, dir, "hello.go", "package main\n\nfunc Hello() string { return \"hello\" }\n")
	gitAdd(t, dir, "hello.go")
	gitCommit(t, dir, "Add hello function")

	writeFile(t, dir, "world.go", "package main\n\nfunc World() string { return \"world\" }\n")
	gitAdd(t, dir, "world.go")
	gitCommit(t, dir, "Add world function")

	return dir, func() {}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func gitAdd(t *testing.T, dir string, paths ...string) {
	t.Helper()
	args := append([]string{"add"}, paths...)
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %s\n%s", err, out)
	}
}

func gitCommit(t *testing.T, dir, msg string) {
	t.Helper()
	cmd := exec.Command("git", "commit", "-m", msg)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %s\n%s", err, out)
	}
}

func gitCheckout(t *testing.T, dir string, args ...string) {
	t.Helper()
	fullArgs := append([]string{"checkout"}, args...)
	cmd := exec.Command("git", fullArgs...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git checkout: %s\n%s", err, out)
	}
}

func TestCurrentBranch(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	branch, err := repo.CurrentBranch()
	if err != nil {
		t.Fatal(err)
	}
	if branch != "feature" {
		t.Errorf("got branch %q, want %q", branch, "feature")
	}
}

func TestMergeBase(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	mb, err := repo.MergeBase("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(mb) != 40 {
		t.Errorf("expected 40-char SHA, got %q", mb)
	}

	// Merge base should match the main branch HEAD
	mainSHA, _ := repo.ResolveSHA("main")
	if mb != mainSHA {
		t.Errorf("merge base %q != main SHA %q", mb, mainSHA)
	}
}

func TestCommitsBetween(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	commits, err := repo.CommitsBetween("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 2 {
		t.Fatalf("expected 2 commits, got %d", len(commits))
	}
	// Oldest first
	if commits[0].Subject != "Add hello function" {
		t.Errorf("first commit subject = %q, want %q", commits[0].Subject, "Add hello function")
	}
	if commits[1].Subject != "Add world function" {
		t.Errorf("second commit subject = %q, want %q", commits[1].Subject, "Add world function")
	}
}

func TestDiffBetween(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	diff, err := repo.DiffBetween("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "hello.go") {
		t.Error("diff should contain hello.go")
	}
	if !strings.Contains(diff, "world.go") {
		t.Error("diff should contain world.go")
	}
}

func TestDiffForCommit(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	commits, _ := repo.CommitsBetween("main", "feature")
	// Get diff for "Add hello function" (first in list, oldest)
	diff, err := repo.DiffForCommit(commits[0].SHA)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(diff, "hello.go") {
		t.Error("diff should contain hello.go")
	}
	if strings.Contains(diff, "world.go") {
		t.Error("diff should not contain world.go")
	}
}

func TestChangedFiles(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	files, err := repo.ChangedFiles("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 changed files, got %d", len(files))
	}

	paths := make(map[string]string)
	for _, f := range files {
		paths[f.Path] = f.Status
	}
	if paths["hello.go"] != "A" {
		t.Errorf("hello.go status = %q, want A", paths["hello.go"])
	}
	if paths["world.go"] != "A" {
		t.Errorf("world.go status = %q, want A", paths["world.go"])
	}
}

func TestResolveSHA(t *testing.T) {
	dir, cleanup := setupTestRepo(t)
	defer cleanup()

	repo := NewRepo(dir)
	sha, err := repo.ResolveSHA("feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(sha) != 40 {
		t.Errorf("expected 40-char SHA, got %q (len=%d)", sha, len(sha))
	}
}
