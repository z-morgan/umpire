# Teach Claude: Accumulate Review Feedback for Config Training

## Problem

When a developer reviews AI-generated code with Umpire and leaves feedback, that feedback is saved as a JSON file but never feeds back into Claude's behavior. The same mistakes repeat because Claude's configuration artifacts (CLAUDE.md, rules, skills, agents) don't evolve based on review patterns.

## Solution

Add a lightweight feedback accumulation pipeline to Umpire:

1. After each review submission, ask the user if they want to record the feedback for future Claude training.
2. If yes, capture a snapshot (review + diff + repo context) to `~/.umpire/feedback/`.
3. On the 5th opt-in (and every subsequent one), offer to generate a self-contained prompt the user can paste into a Claude Code session.
4. That prompt instructs Claude to read all accumulated snapshots, identify behavioral patterns, propose config changes, and clean up.

Umpire stays fully decoupled from Claude — it captures data and generates text. No API calls, no subprocess spawning.

## Data Model

### Feedback Snapshot (`~/.umpire/feedback/snapshot-YYYYMMDD-HHMMSS.json`)

```json
{
  "version": 1,
  "created_at": "2026-04-21T18:30:00Z",
  "repo_path": "/Users/zmorgan/zm_apps/myproject",
  "base_ref": "main",
  "head_ref": "feature-x",
  "base_sha": "abc123",
  "head_sha": "def456",
  "diff": "diff --git a/foo.go ...",
  "review": {
    "summary": "Several naming issues and a missing error check",
    "comments": [
      {
        "file": "internal/handler.go",
        "line_start": 42,
        "side": "right",
        "body": "Use descriptive names — 'req' doesn't tell me what kind of request",
        "diff_hunk": "@@ -40,3 +40,5 ..."
      }
    ]
  }
}
```

The snapshot captures everything the prompt needs. Claude config artifacts are NOT included in the snapshot — the generated prompt tells Claude to read them live, since they may have changed between recording and analysis.

## UX Flow

The current post-submission behavior: submit bar is replaced with "Review saved to <path> / Server shutting down..."

New behavior — the submit bar becomes a multi-step inline flow:

### Step 1: Review saved
```
Review saved to .umpire/reviews/review-20260421-183000.json

Record this feedback to improve future Claude sessions?   [Yes]  [No thanks]
```

### Step 2a: User clicks "No thanks"
Server shuts down as before.

### Step 2b: User clicks "Yes" (count < 5 after recording)
```
Feedback recorded (3 of 5 until analysis is available).
Server shutting down...
```

### Step 2c: User clicks "Yes" (count >= 5 after recording)
```
Feedback recorded — 5 snapshots available.

Generate a prompt to analyze your feedback and propose Claude config updates?

[Copy Prompt to Clipboard]   [Not now]
```

Clicking either button shuts down the server. "Copy Prompt" copies the generated prompt text to the clipboard first.

## The Generated Prompt

The prompt is a self-contained instruction block the user pastes into `claude` (a raw Claude Code session). It tells Claude to:

1. Read all JSON files in `~/.umpire/feedback/`
2. For each snapshot, understand the diff and the review comments
3. Classify feedback as **generalizable preferences** vs **one-off corrections** — only generalizable patterns should produce config changes
4. Look for recurring themes: naming conventions, error handling style, architectural preferences, testing expectations, code organization
5. Read the user's existing **user-level** Claude config artifacts:
   - `~/.claude/CLAUDE.md` (global instructions)
   - `~/.claude/settings.json` (global settings)
6. Propose specific additions or edits to those files that would address the identified patterns — without duplicating or contradicting existing rules. Changes are scoped to user-level config only; project-level files (per-repo `.claude/CLAUDE.md`, `.cursorrules`, project settings, commands, agents) are not modified
7. Present the proposed changes for user approval before applying
8. After changes are applied, delete the processed snapshot files from `~/.umpire/feedback/`

The prompt is generated server-side (Go) so it can embed the current snapshot count.

## Implementation Plan

### Step 1: Feedback package — data model and storage

Create `internal/feedback/` with:

- **`snapshot.go`**: `Snapshot` struct matching the JSON schema above, plus a `SubmitRequest` struct for the API input.
- **`store.go`**: `Store` struct with:
  - `Dir` field (`~/.umpire/feedback/`)
  - `Save(s *Snapshot) (path string, err error)` — writes timestamped JSON
  - `Count() (int, error)` — counts existing snapshot files
- **`prompt.go`**: `GeneratePrompt(count int) string` — builds the analysis prompt text targeting user-level config (`~/.claude/CLAUDE.md`, `~/.claude/settings.json`)

Files: `internal/feedback/snapshot.go`, `internal/feedback/store.go`, `internal/feedback/prompt.go`
Tests: `internal/feedback/store_test.go`

### Step 2: API endpoints

Add two new handlers to `internal/server/handlers.go`:

- **`POST /api/record-feedback`**: Accepts `{ "diff": "...", "review": {...} }`. Builds a `Snapshot` from the request body + `ReviewContext` metadata (refs, SHAs, repo path). Saves via feedback store. Returns `{ "count": 5, "threshold_reached": true }`.
- **`GET /api/feedback-prompt`**: Generates and returns the prompt text as `{ "prompt": "..." }`.

Also add a **`POST /api/shutdown`** endpoint so the frontend can explicitly trigger server shutdown after the feedback flow completes (replacing the current auto-shutdown-on-submit behavior).

Wire the feedback store into `ReviewContext` and update `cmd/root.go` to:
- Resolve `~/.umpire/feedback/` and create the feedback `Store`
- Pass it into `ReviewContext`
- Change shutdown logic: review submission no longer auto-triggers shutdown; instead the `/api/shutdown` endpoint sends to `submitCh`

Files: `internal/server/handlers.go`, `cmd/root.go`

### Step 3: Frontend — post-submission feedback flow

Modify `submitReview()` in `web/static/js/app.js`:

After the review is saved successfully, instead of immediately showing "Server shutting down...", render the feedback question inline in the submit bar. Handle the multi-step flow:

1. Show "Record feedback?" question with Yes/No buttons
2. On "No" → POST `/api/shutdown`, show "Server shutting down..."
3. On "Yes" → POST `/api/record-feedback` with the diff + review data
4. If `threshold_reached` is false → show count message, then POST `/api/shutdown`
5. If `threshold_reached` is true → show prompt offer with "Copy Prompt" / "Not now" buttons
6. On "Copy Prompt" → GET `/api/feedback-prompt`, copy to clipboard, then POST `/api/shutdown`
7. On "Not now" → POST `/api/shutdown`

The diff data needs to be available at submission time. Currently the frontend fetches the diff for display but doesn't store it. We'll cache the full diff (base..head) when it's first loaded and keep it in `App` state so it's available for the feedback snapshot.

Files: `web/static/js/app.js`, `web/static/css/app.css` (for feedback dialog styling)

## Design Decisions

**Why `~/.umpire/` and not per-repo?** Feedback accumulates across all projects because the prompt only targets user-level config (`~/.claude/CLAUDE.md`, `~/.claude/settings.json`). A naming preference expressed in one repo applies globally. The `repo_path` field in each snapshot provides context for understanding the review but doesn't drive where changes are written.

**Why only user-level config?** Project-level files (per-repo `.claude/CLAUDE.md`, `.cursorrules`, commands, agents) are curated per-project and reflect project-specific conventions. Review feedback accumulated across repos produces cross-cutting preferences — those belong in the user's global config. Modifying project-level files from cross-project feedback would be noisy and potentially wrong. This can be revisited in a future version if there's demand for project-scoped analysis.

**Why a threshold of 5?** Enough data points to identify patterns without being so many that the user forgets about the feature. Hardcoded for simplicity in v1.

**Why a copy-paste prompt instead of spawning Claude directly?** Keeps Umpire decoupled — no dependency on Claude being installed, no API keys, no subprocess management. The user controls when and where the prompt runs. They can also edit it before pasting.

**Why not record the diff as part of the review JSON?** The review is already saved to `.umpire/reviews/` and serves a different purpose (human-readable record). The feedback snapshot is a separate artifact optimized for AI consumption with different data (includes full diff, repo path, etc.).

**Why explicit /api/shutdown instead of auto-shutdown?** The current flow has the server shutting down immediately after review save. With the feedback dialog, we need the server to stay alive for additional API calls. Making shutdown explicit keeps the control flow clean.
