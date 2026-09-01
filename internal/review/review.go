package review

import "time"

// CommitMessageEditWrapInstruction is the agent-facing note emitted alongside
// commit message edits. It covers only the body: hard-wrap it at 72 columns,
// the standard git convention. It deliberately says nothing about the subject.
const CommitMessageEditWrapInstruction = "When applying these commit message edits, hard-wrap the body at 72 columns, per standard git commit conventions."

// Review represents a complete code review with comments.
type Review struct {
	Version                       int                 `json:"version"`
	BaseRef                       string              `json:"base_ref"`
	HeadRef                       string              `json:"head_ref"`
	BaseSHA                       string              `json:"base_sha"`
	HeadSHA                       string              `json:"head_sha"`
	CreatedAt                     time.Time           `json:"created_at"`
	Summary                       string              `json:"summary"`
	Comments                      []Comment           `json:"comments"`
	CommitMessageEdits            []CommitMessageEdit `json:"commit_message_edits,omitempty"`
	CommitMessageEditInstructions string              `json:"commit_message_edit_instructions,omitempty"`
}

// CommitMessageEdit captures a user's rewrite of a commit message during review.
// An implementing agent reads these to rewrite the full commit message, hard-
// wrapping the body at 72 columns per standard git convention; the original
// subject and body are kept for context.
type CommitMessageEdit struct {
	SHA             string `json:"sha"`
	OriginalSubject string `json:"original_subject"`
	OriginalBody    string `json:"original_body"`
	EditedSubject   string `json:"edited_subject"`
	EditedBody      string `json:"edited_body"`
}

// Comment represents a single inline comment on a diff.
//
// The anchoring contract for an agent consuming a saved review:
//   - CommitSHA empty  => LineStart is relative to the head revision, i.e.
//     the comment was authored against the full base..head diff.
//   - CommitSHA set     => LineStart is relative to that commit's diff, and
//     DiffHunk is the authoritative locator, because a later commit may have
//     shifted the same line in the head revision.
//
// Trust LineStart directly only in the empty-CommitSHA case; otherwise resolve
// the line through DiffHunk.
type Comment struct {
	ID        string `json:"id"`
	File      string `json:"file"`
	CommitSHA string `json:"commit_sha,omitempty"`
	LineStart int    `json:"line_start"`
	LineEnd   int    `json:"line_end"`
	Side      string `json:"side"`
	Body      string `json:"body"`
	// DiffHunk is a few lines of surrounding context, each carrying its diff
	// prefix (+/-/space). It is the authoritative locator when CommitSHA is set.
	DiffHunk string `json:"diff_hunk"`
}

// SubmitRequest is the JSON body sent by the frontend to submit a review.
type SubmitRequest struct {
	Summary            string              `json:"summary"`
	Comments           []Comment           `json:"comments"`
	CommitMessageEdits []CommitMessageEdit `json:"commit_message_edits,omitempty"`
}
