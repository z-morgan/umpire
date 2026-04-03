package review

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store handles saving reviews to disk.
type Store struct {
	Dir string // typically ".umpire/reviews" relative to repo root
}

// NewStore creates a Store that writes to the given directory.
func NewStore(repoDir string) *Store {
	return &Store{
		Dir: filepath.Join(repoDir, ".umpire", "reviews"),
	}
}

// Save writes a review to a timestamped JSON file and returns the file path.
func (s *Store) Save(r *Review) (string, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", fmt.Errorf("creating review directory: %w", err)
	}

	r.CreatedAt = time.Now().UTC()
	r.Version = 1

	filename := fmt.Sprintf("review-%s.json", r.CreatedAt.Format("20060102-150405"))
	path := filepath.Join(s.Dir, filename)

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling review: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("writing review file: %w", err)
	}

	return path, nil
}
