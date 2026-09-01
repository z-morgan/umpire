package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zmorgan/umpire/internal/feedback"
	"github.com/zmorgan/umpire/internal/git"
	"github.com/zmorgan/umpire/internal/review"
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
	ts, _ := setupTestServerWithContext(t)
	return ts
}

func setupTestServerWithContext(t *testing.T) (*httptest.Server, *ReviewContext) {
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
		Repo:          repo,
		BaseRef:       "main",
		HeadRef:       "feature",
		BaseSHA:       baseSHA,
		HeadSHA:       headSHA,
		MergeBase:     mergeBase,
		Store:         &review.Store{Dir: t.TempDir()},
		FeedbackStore: &feedback.Store{Dir: t.TempDir()},
	}

	mux := http.NewServeMux()
	RegisterAPI(mux, rc)
	return httptest.NewServer(mux), rc
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

func TestHandleReviewPersistsCommitMessageEdits(t *testing.T) {
	ts, rc := setupTestServerWithContext(t)
	defer ts.Close()

	body := review.SubmitRequest{
		Summary: "Reword the commit message.",
		CommitMessageEdits: []review.CommitMessageEdit{
			{
				SHA:             "deadbeef",
				OriginalSubject: "wip",
				OriginalBody:    "",
				EditedSubject:   "Add hello function",
				EditedBody:      "Returns a friendly greeting.",
			},
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(ts.URL+"/api/review", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Find the review file the handler just wrote and verify the edits persisted.
	matches, err := filepath.Glob(filepath.Join(rc.Store.Dir, "review-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 review file, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var rev review.Review
	if err := json.Unmarshal(data, &rev); err != nil {
		t.Fatal(err)
	}
	if len(rev.CommitMessageEdits) != 1 {
		t.Fatalf("len(CommitMessageEdits) = %d, want 1", len(rev.CommitMessageEdits))
	}
	edit := rev.CommitMessageEdits[0]
	if edit.SHA != "deadbeef" {
		t.Errorf("SHA = %q, want deadbeef", edit.SHA)
	}
	if edit.EditedSubject != "Add hello function" {
		t.Errorf("EditedSubject = %q, want %q", edit.EditedSubject, "Add hello function")
	}
	if edit.OriginalSubject != "wip" {
		t.Errorf("OriginalSubject = %q, want %q", edit.OriginalSubject, "wip")
	}
	if rev.CommitMessageEditInstructions != review.CommitMessageEditWrapInstruction {
		t.Errorf("CommitMessageEditInstructions = %q, want %q", rev.CommitMessageEditInstructions, review.CommitMessageEditWrapInstruction)
	}
}

func TestHandleReviewOmitsInstructionsWithoutEdits(t *testing.T) {
	ts, rc := setupTestServerWithContext(t)
	defer ts.Close()

	body := review.SubmitRequest{Summary: "Looks good, no message changes."}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(ts.URL+"/api/review", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	matches, err := filepath.Glob(filepath.Join(rc.Store.Dir, "review-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 review file, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	// With no commit-message edits, the instruction key should be absent
	// entirely (omitempty), not just empty.
	if strings.Contains(string(data), "commit_message_edit_instructions") {
		t.Errorf("review JSON should omit the instruction key when there are no edits:\n%s", data)
	}
}

func TestHandleRecordFeedbackPersistsSnapshot(t *testing.T) {
	ts, rc := setupTestServerWithContext(t)
	defer ts.Close()

	body := feedback.SubmitRequest{
		Diff: "diff --git a/hello.go b/hello.go\n",
		Review: feedback.Review{
			Summary: "Naming could be clearer.",
		},
	}
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Post(ts.URL+"/api/record-feedback", "application/json", bytes.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}

	// Find the snapshot file the handler just wrote and verify it captured the review.
	matches, err := filepath.Glob(filepath.Join(rc.FeedbackStore.Dir, "snapshot-*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 1 {
		t.Fatalf("expected 1 snapshot file, got %d", len(matches))
	}

	data, err := os.ReadFile(matches[0])
	if err != nil {
		t.Fatal(err)
	}
	var snap feedback.Snapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		t.Fatal(err)
	}
	if snap.Review.Summary != "Naming could be clearer." {
		t.Errorf("Review.Summary = %q, want %q", snap.Review.Summary, "Naming could be clearer.")
	}
	if snap.Diff != "diff --git a/hello.go b/hello.go\n" {
		t.Errorf("Diff = %q, want the submitted diff", snap.Diff)
	}
}
