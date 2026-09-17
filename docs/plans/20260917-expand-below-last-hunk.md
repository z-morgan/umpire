# Expand Below the Last Hunk

## Problem

Umpire's diff view lets you click the blue hunk-header bars to pull in the
truncated context around a change, but only the context *between* rendered
regions is reachable. The tail of every file — everything after the last hunk —
has no bar, so there is no way to read the code below the final change.

`DiffExpander.attach()` (`web/static/js/diff-expander.js`) only wires up rows
diff2html already emitted: the `.d2h-info` hunk headers. Each expands *upward*,
filling the gap between the previous rendered line and the `@@` header's start
line. Git emits no hunk header after the last hunk, so there is nothing at the
bottom to hang a bar on.

Two pieces are missing:

1. **A row to click.** No `.d2h-info` row exists after the last hunk, so one has
   to be synthesized and appended to the file's `tbody`.
2. **A way to know where the file ends.** `computeGap` learns a gap's end by
   parsing `@@ -a,b +c,d @@`. At the bottom of the file there is no header to
   parse, so the server has to report the file's line count.

## Solution

A blue bar at the bottom of each file's rendered diff, matching the existing
hunk-header bars, that expands downward in batches until it reaches the end of
the file and then removes itself. Files whose diff already runs to EOF — the
common "appended to the end of the file" case — get no bar.

### Step 1 — Refactor: consume a gap from either end

`handleExpand` hardcodes consuming from the *bottom* of the gap: it fetches
`[max(start, end - BATCH + 1), end]` and walks `gapEnd` backwards. A trailing
bar needs the mirror image — fetch `[start, min(end, start + BATCH - 1)]` and
walk `gapStart` forwards. Both insert their rows *before* the clicked row, so
only the fetch window and the gap bookkeeping differ.

Split that window-and-bookkeeping decision out of `handleExpand` so the
direction is a parameter, keeping the fetch/insert/highlight body shared.
Existing hunk-header bars behave identically afterward. No behavior change.

### Step 2 — Backend: report file line counts at a ref

New endpoint:

```
GET /api/file-line-counts?ref=<sha>&path=a&path=b  ->  {"a": 120, "b": 43}
```

Registered alongside the other routes in `RegisterAPI`
(`internal/server/handlers.go`). It calls `Repo.ShowFile` per path and counts
lines, **omitting** any path it cannot read. Deleted and binary files then
simply get no bar, with no special-casing on the client.

Batched rather than one request per file: the diff view renders every file at
once, so this is a single round trip per diff render.

Note that `Repo.run` trims trailing whitespace, so the count excludes trailing
blank lines. That matches what `/api/file-lines` already returns, so expansion
stays self-consistent.

Tests: `TestHandleFileLineCounts` covering a real path, a missing path (omitted
from the response), and several paths in one request.

### Step 3 — Frontend: the trailing bar

Add `API.getFileLineCounts` to `api.js`. In `DiffExpander.attach()`, after
wiring the hunk-header rows, for each `.d2h-file-wrapper`:

- resolve the file path and look up its line count; skip the file when the
  count is absent (deleted, binary, unreadable);
- find the last rendered new-side line by running the existing
  `findPrevLineNumber` backwards from the `tbody`'s last row — it already skips
  comment and comment-form rows;
- when `lastLine < total`, append a bar row carrying
  `data-gap-start = lastLine + 1` and `data-gap-end = total`, wired to the
  downward direction from step 1.

The bar mirrors diff2html's own info-row markup (`td.d2h-code-linenumber.d2h-info`
plus `td.d2h-info > div.d2h-code-line`) so it picks up the blue `.d2h-info`
background and the `tr.d2h-expandable` hover already in `app.css`. Where a hunk
header shows its `@@ ... @@` text, the trailing bar shows a downward affordance
instead; the CSS delta is a modifier class for that label.

### Step 4 — Manual verification

There is no JS test harness in the repo, so step 3 is verified by running
umpire against a branch containing:

- a file modified mid-way — bar appears, expands in batches of 20, disappears
  on reaching EOF;
- a file modified at its end — no bar;
- a deleted file — no bar;
- a file with a comment on its last line — bar sits below the comment.

## Pre-existing issues found while planning

Filed separately rather than fixed here:

- **Expanded context lines aren't commentable.** `buildContextRow` creates rows
  with `.d2h-code-linenumber`, but `DiffView.attachLineClickHandlers` only runs
  at render time, so inserted lines never get click handlers. Already true
  today; the trailing bar makes it more noticeable.
- **The expander breaks on renamed files.** It reads the path from
  `.d2h-file-name`, which diff2html renders as `old -> new` for a rename, so the
  lookup cannot resolve. Related to the "Files changed" sidebar work. The new
  code fails soft here: no count means no bar.
