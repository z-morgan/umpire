package feedback

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Store handles saving feedback snapshots to disk.
type Store struct {
	Dir string // typically ~/.umpire/feedback/
}

// NewStore creates a Store that writes to ~/.umpire/feedback/.
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	return &Store{Dir: filepath.Join(home, ".umpire", "feedback")}, nil
}

// Save writes a snapshot to a timestamped JSON file and returns the file path.
func (s *Store) Save(snap *Snapshot) (string, error) {
	if err := os.MkdirAll(s.Dir, 0o755); err != nil {
		return "", fmt.Errorf("creating feedback directory: %w", err)
	}

	snap.CreatedAt = time.Now().UTC()
	snap.Version = 1

	filename := fmt.Sprintf("snapshot-%s.json", snap.CreatedAt.Format("20060102-150405"))
	path := filepath.Join(s.Dir, filename)

	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("writing snapshot file: %w", err)
	}

	return path, nil
}

// Count returns the number of snapshot files in the store directory.
func (s *Store) Count() (int, error) {
	matches, err := filepath.Glob(filepath.Join(s.Dir, "snapshot-*.json"))
	if err != nil {
		return 0, fmt.Errorf("counting snapshots: %w", err)
	}
	return len(matches), nil
}
