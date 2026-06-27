package review

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStoreSave(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Dir: filepath.Join(dir, ".umpire", "reviews")}

	r := &Review{
		BaseRef: "main",
		HeadRef: "feature",
		BaseSHA: "abc123",
		HeadSHA: "def456",
		Summary: "Looks good overall",
		Comments: []Comment{
			{
				ID:        "c1",
				File:      "hello.go",
				LineStart: 10,
				LineEnd:   10,
				Side:      "right",
				Body:      "Consider renaming this variable",
				DiffHunk:  "func Hello() string {",
			},
		},
	}

	path, err := store.Save(r)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.HasPrefix(path, store.Dir) {
		t.Errorf("path %q should be under %q", path, store.Dir)
	}
	if !strings.HasSuffix(path, ".json") {
		t.Errorf("path %q should end with .json", path)
	}

	// Verify file contents
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Review
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.Version != 1 {
		t.Errorf("version = %d, want 1", loaded.Version)
	}
	if loaded.Summary != "Looks good overall" {
		t.Errorf("summary = %q, want %q", loaded.Summary, "Looks good overall")
	}
	if len(loaded.Comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(loaded.Comments))
	}
	if loaded.Comments[0].Body != "Consider renaming this variable" {
		t.Errorf("comment body = %q", loaded.Comments[0].Body)
	}
	if loaded.CreatedAt.IsZero() {
		t.Error("created_at should be set")
	}
}

func TestStoreSaveRoundTripsCommitMessageEdits(t *testing.T) {
	dir := t.TempDir()
	store := &Store{Dir: filepath.Join(dir, ".umpire", "reviews")}

	r := &Review{
		BaseRef: "main",
		HeadRef: "feature",
		Summary: "Reword the commit",
		CommitMessageEdits: []CommitMessageEdit{
			{
				SHA:             "def456",
				OriginalSubject: "wip",
				OriginalBody:    "",
				EditedSubject:   "Add hello function",
				EditedBody:      "Returns a friendly greeting.",
			},
		},
	}

	path, err := store.Save(r)
	if err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Review
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatal(err)
	}

	if len(loaded.CommitMessageEdits) != 1 {
		t.Fatalf("len(CommitMessageEdits) = %d, want 1", len(loaded.CommitMessageEdits))
	}
	edit := loaded.CommitMessageEdits[0]
	if edit.SHA != "def456" {
		t.Errorf("edit.SHA = %q, want def456", edit.SHA)
	}
	if edit.OriginalSubject != "wip" {
		t.Errorf("edit.OriginalSubject = %q, want %q", edit.OriginalSubject, "wip")
	}
	if edit.EditedSubject != "Add hello function" {
		t.Errorf("edit.EditedSubject = %q, want %q", edit.EditedSubject, "Add hello function")
	}
	if edit.EditedBody != "Returns a friendly greeting." {
		t.Errorf("edit.EditedBody = %q, want %q", edit.EditedBody, "Returns a friendly greeting.")
	}
}

func TestStoreSaveCreatesDirectory(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "deep", "nested", "reviews")
	store := &Store{Dir: nested}

	r := &Review{
		BaseRef: "main",
		HeadRef: "feature",
		Summary: "Test",
	}

	path, err := store.Save(r)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Errorf("review file should exist: %v", err)
	}
}
