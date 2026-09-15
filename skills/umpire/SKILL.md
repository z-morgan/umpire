---
name: umpire
description: Run a local code review in the browser with umpire and act on the review the human submits. Use when the user invokes /umpire, or asks for a review of the current branch, to look over the diff before merging, or to open the changes in a review UI.
---

# Umpire

Umpire is a local code review tool. It serves a browser UI showing the diff
between a base branch and the current branch, the human comments on it and
submits, and the review is saved to `.umpire/reviews/` as JSON.

Umpire blocks until the human is finished with the UI. That is the whole
mechanism this skill relies on: launch it in the background, end your turn, and
the harness will re-invoke you when the process exits.

## Launching a review

Run the binary with the Bash tool and `run_in_background: true`:

```
umpire
```

Pass any arguments given to `/umpire` straight through. `umpire --base develop`
and `umpire --head feature/auth` are the common ones; `umpire --help` lists the
rest.

Umpire opens the browser itself, so there is nothing to click through on your
side.

## Then stop

**End your turn immediately after launching.** Tell the user their review is
open and that you will pick it up when they submit, and say nothing else.

This is the part that is easy to get wrong. The review takes as long as it takes
— minutes, sometimes longer — and there is no version of waiting for it that
works. Specifically, do not:

- Poll the background task output to see whether the review has been submitted.
- Sleep, retry, or otherwise loop until the process exits.
- Watch `.umpire/reviews/` for a new file.
- Start on other work in the same turn, or ask the user what to do next.

You will be re-invoked automatically when umpire exits. Waiting costs a turn and
buys nothing.

You may read the background task output **once**, right after launching, to
report the URL umpire printed. If the banner hasn't appeared yet, leave it out
and end your turn anyway. Do not read a second time.

## When umpire exits

The harness wakes you with the process's output. Umpire prints the path it saved
the review to:

```
umpire: review saved to .umpire/reviews/review-20260915-142233.json
```

Read that file and act on the review.
