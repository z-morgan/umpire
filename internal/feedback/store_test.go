package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/zmorgan/umpire/internal/review"
)

func TestSave(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Dir: dir}

	snap := &Snapshot{
		RepoPath: "/tmp/myproject",
		BaseRef:  "main",
		HeadRef:  "feature-x",
		BaseSHA:  "abc123",
		HeadSHA:  "def456",
		Diff:     "diff --git a/foo.go b/foo.go\n",
		Review: Review{
			Summary: "Naming issues",
			Comments: []review.Comment{
				{
					ID:        "c1",
					File:      "foo.go",
					LineStart: 10,
					Side:      "right",
					Body:      "Use a descriptive name here",
				},
			},
		},
	}

	path, err := store.Save(snap)
	if err != nil {
		t.Fatalf("Save() error: %v", err)
	}

	if filepath.Dir(path) != dir {
		t.Errorf("Save() wrote to %s, want dir %s", path, dir)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading saved file: %v", err)
	}

	var loaded Snapshot
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("unmarshaling saved file: %v", err)
	}

	if loaded.Version != 1 {
		t.Errorf("Version = %d, want 1", loaded.Version)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("CreatedAt is zero, want non-zero")
	}
	if loaded.RepoPath != "/tmp/myproject" {
		t.Errorf("RepoPath = %q, want %q", loaded.RepoPath, "/tmp/myproject")
	}
	if loaded.Review.Summary != "Naming issues" {
		t.Errorf("Review.Summary = %q, want %q", loaded.Review.Summary, "Naming issues")
	}
	if len(loaded.Review.Comments) != 1 {
		t.Fatalf("len(Comments) = %d, want 1", len(loaded.Review.Comments))
	}
	if loaded.Review.Comments[0].Body != "Use a descriptive name here" {
		t.Errorf("Comment body = %q, want %q", loaded.Review.Comments[0].Body, "Use a descriptive name here")
	}
}

func TestCount(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Dir: dir}

	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d for empty dir, want 0", count)
	}

	// Create snapshot files directly to avoid same-second timestamp collisions.
	for i := 0; i < 3; i++ {
		name := fmt.Sprintf("snapshot-20260422-12000%d.json", i)
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{}"), 0o644); err != nil {
			t.Fatalf("creating test file %d: %v", i, err)
		}
	}

	count, err = store.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 3 {
		t.Errorf("Count() = %d, want 3", count)
	}
}

func TestCountNonexistentDir(t *testing.T) {
	store := &Store{Dir: filepath.Join(t.TempDir(), "nonexistent")}

	count, err := store.Count()
	if err != nil {
		t.Fatalf("Count() error: %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d for nonexistent dir, want 0", count)
	}
}
