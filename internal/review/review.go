package review

import "time"

// Review represents a complete code review with comments.
type Review struct {
	Version            int                 `json:"version"`
	BaseRef            string              `json:"base_ref"`
	HeadRef            string              `json:"head_ref"`
	BaseSHA            string              `json:"base_sha"`
	HeadSHA            string              `json:"head_sha"`
	CreatedAt          time.Time           `json:"created_at"`
	Summary            string              `json:"summary"`
	Comments           []Comment           `json:"comments"`
	CommitMessageEdits []CommitMessageEdit `json:"commit_message_edits,omitempty"`
}

// CommitMessageEdit captures a user's rewrite of a commit message during review.
// An implementing agent reads these to rewrite the full commit message; the
// original subject and body are kept for context.
type CommitMessageEdit struct {
	SHA             string `json:"sha"`
	OriginalSubject string `json:"original_subject"`
	OriginalBody    string `json:"original_body"`
	EditedSubject   string `json:"edited_subject"`
	EditedBody      string `json:"edited_body"`
}

// Comment represents a single inline comment on a diff.
type Comment struct {
	ID        string `json:"id"`
	File      string `json:"file"`
	CommitSHA string `json:"commit_sha,omitempty"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Side      string `json:"side"`
	Body      string `json:"body"`
	DiffHunk  string `json:"diff_hunk"`
}

// SubmitRequest is the JSON body sent by the frontend to submit a review.
type SubmitRequest struct {
	Summary            string              `json:"summary"`
	Comments           []Comment           `json:"comments"`
	CommitMessageEdits []CommitMessageEdit `json:"commit_message_edits,omitempty"`
}
