package server

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zmorgan/umpire/internal/git"
)

func setupTestRepo(t *testing.T) *git.Repo {
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

	writeFile(t, dir, "README.md", "# Test\n")
	gitRun(t, dir, "add", ".")
	gitRun(t, dir, "commit", "-m", "Initial commit")

	gitRun(t, dir, "checkout", "-b", "feature")
	writeFile(t, dir, "hello.go", "package main\n\nfunc Hello() string { return \"hello\" }\n")
	gitRun(t, dir, "add", "hello.go")
	gitRun(t, dir, "commit", "-m", "Add hello function")

	return git.NewRepo(dir)
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

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %s\n%s", strings.Join(args, " "), err, out)
	}
}

func setupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo := setupTestRepo(t)

	baseSHA, err := repo.ResolveSHA("main")
	if err != nil {
		t.Fatal(err)
	}
	headSHA, err := repo.ResolveSHA("feature")
	if err != nil {
		t.Fatal(err)
	}
	mergeBase, err := repo.MergeBase(baseSHA, headSHA)
	if err != nil {
		t.Fatal(err)
	}

	rc := &ReviewContext{
		Repo:      repo,
		BaseRef:   "main",
		HeadRef:   "feature",
		BaseSHA:   baseSHA,
		HeadSHA:   headSHA,
		MergeBase: mergeBase,
	}

	mux := http.NewServeMux()
	RegisterAPI(mux, rc)
	return httptest.NewServer(mux)
}

func TestHandleInfo(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/info")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	var info map[string]string
	json.NewDecoder(resp.Body).Decode(&info)

	if info["base_ref"] != "main" {
		t.Errorf("base_ref = %q, want main", info["base_ref"])
	}
	if info["head_ref"] != "feature" {
		t.Errorf("head_ref = %q, want feature", info["head_ref"])
	}
}

func TestHandleCommits(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/commits")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var commits []git.Commit
	json.NewDecoder(resp.Body).Decode(&commits)

	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Subject != "Add hello function" {
		t.Errorf("subject = %q, want %q", commits[0].Subject, "Add hello function")
	}
}

func TestHandleDiff(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/diff")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	diff := string(body)

	if !strings.Contains(diff, "hello.go") {
		t.Error("diff should contain hello.go")
	}
}

func TestHandleDiffForCommit(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	// Get the commit SHA first
	resp, err := http.Get(ts.URL + "/api/commits")
	if err != nil {
		t.Fatal(err)
	}
	var commits []git.Commit
	json.NewDecoder(resp.Body).Decode(&commits)
	resp.Body.Close()

	resp, err = http.Get(ts.URL + "/api/diff?commit=" + commits[0].SHA)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	diff := string(body)

	if !strings.Contains(diff, "hello.go") {
		t.Error("diff should contain hello.go")
	}
}

func TestHandleFiles(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/api/files")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var files []git.ChangedFile
	json.NewDecoder(resp.Body).Decode(&files)

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "hello.go" {
		t.Errorf("path = %q, want hello.go", files[0].Path)
	}
}
