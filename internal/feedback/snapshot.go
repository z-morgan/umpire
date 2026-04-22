package feedback

import (
	"time"

	"github.com/zmorgan/umpire/internal/review"
)

// Snapshot captures a review with its diff and repo context for later analysis.
type Snapshot struct {
	Version   int       `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	RepoPath  string    `json:"repo_path"`
	BaseRef   string    `json:"base_ref"`
	HeadRef   string    `json:"head_ref"`
	BaseSHA   string    `json:"base_sha"`
	HeadSHA   string    `json:"head_sha"`
	Diff      string    `json:"diff"`
	Review    Review    `json:"review"`
}

// Review is the review portion of a feedback snapshot.
type Review struct {
	Summary  string           `json:"summary"`
	Comments []review.Comment `json:"comments"`
}

// SubmitRequest is the JSON body the frontend POSTs to record feedback.
type SubmitRequest struct {
	Diff   string `json:"diff"`
	Review Review `json:"review"`
}
