package feedback

import (
	"time"

	"github.com/zmorgan/umpire/internal/review"
)

// Snapshot captures a review with its diff and repo context for later analysis.
type Snapshot struct {
	Version            int                 `json:"version"`
	CreatedAt          time.Time           `json:"created_at"`
	RepoPath           string              `json:"repo_path"`
	BaseRef            string              `json:"base_ref"`
	HeadRef            string              `json:"head_ref"`
	BaseSHA            string              `json:"base_sha"`
	HeadSHA            string              `json:"head_sha"`
	Diff               string              `json:"diff"`
	Review             Review              `json:"review"`
	CommitMessageEdits []CommitMessageEdit `json:"commit_message_edits,omitempty"`
}

// Review is the review portion of a feedback snapshot.
type Review struct {
	Summary  string           `json:"summary"`
	Comments []review.Comment `json:"comments"`
}

// CommitMessageEdit captures a user's rewrite of a commit message during review.
type CommitMessageEdit struct {
	SHA             string `json:"sha"`
	OriginalSubject string `json:"original_subject"`
	OriginalBody    string `json:"original_body"`
	EditedSubject   string `json:"edited_subject"`
	EditedBody      string `json:"edited_body"`
}

// SubmitRequest is the JSON body the frontend POSTs to record feedback.
type SubmitRequest struct {
	Diff               string              `json:"diff"`
	Review             Review              `json:"review"`
	CommitMessageEdits []CommitMessageEdit `json:"commit_message_edits"`
}
