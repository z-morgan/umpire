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

### Check there's something to review first

If the head ref has no commits ahead of the base, umpire opens a UI with an
empty diff in it, which wastes the user's trip to the browser:

```
git rev-list --count <base>..<head>
```

A count of zero means say so and stop. Don't launch. Usually it means the base
is wrong — the branch was cut from `develop` and the default base is `main`, or
the work is still uncommitted in the working tree.

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

### If the user changes their mind

A review that is never submitted never exits, so the background task would sit
there indefinitely. If the user says to cancel, forget it, or moves on to
something else, kill the task. Don't leave it running on the chance they come
back to it.

## Finding the review

The harness wakes you with the process's output when umpire exits. Umpire prints
the path it saved the review to:

```
umpire: review saved to .umpire/reviews/review-20260915-142233.json
```

If that line isn't in the output, take the newest file in `.umpire/reviews/`
instead.

## Reading the review

The file is JSON:

```json
{
  "base_ref": "main",
  "head_ref": "feature/auth",
  "base_sha": "...",
  "head_sha": "...",
  "summary": "Overall thoughts on the branch.",
  "comments": [
    {
      "file": "internal/auth/token.go",
      "commit_sha": "a1b2c3d",
      "line_start": 42,
      "line_end": 42,
      "side": "right",
      "body": "This should return an error instead of panicking.",
      "diff_hunk": "..."
    }
  ],
  "commit_message_edits": [],
  "commit_message_edit_instructions": "..."
}
```

`summary` is the review as a whole and often carries the most important
feedback. Read it before the inline comments, not after.

### Anchoring a comment to a line

`line_start` does not mean the same thing in every comment, and getting this
wrong edits the wrong line:

- **`commit_sha` is empty** — the comment was authored against the full
  `base..head` diff, so `line_start` refers to the head revision. Trust it
  directly.
- **`commit_sha` is set** — `line_start` is relative to *that commit's* diff.
  `diff_hunk` is the authoritative locator, because a later commit may have
  shifted the same line in the head revision. Resolve the line by finding the
  `diff_hunk` context in the file rather than jumping to `line_start`.

`side` is `"right"` for the post-change version of the line and `"left"` for the
pre-change version.

### Commit message edits

`commit_message_edits` holds rewrites of commit messages: each carries the
commit's `sha`, the original subject and body for context, and the edited
subject and body to apply. When the array is non-empty,
`commit_message_edit_instructions` carries a note to follow — currently that the
body hard-wraps at 72 columns, per standard git convention.

## Acting on the review

### Questions come first

**If the review asks questions, answer them and stop. Change no code until the
user has replied.**

Review comments are not uniformly change orders. Some are questions, some are
disagreements, some are thinking out loud. "Why did you do it this way?" is a
request for an explanation, and an agent that reads it as an instruction
produces a confident non-answer and rewrites code that was fine.

If some comments are questions and others are clear change orders, still stop.
The answers may change what the rest of the work should be.

### Otherwise, amend the commit each comment landed on

A review comment is about the commit that introduced the line, so fix it there.
Appending fixups to the tip leaves the branch reading as a sequence of mistakes
and corrections, when it could read as though the feedback had been incorporated
the first time. Every comment already knows its commit.

**First check whether the branch has been pushed:**

```
git rev-parse --abbrev-ref @{upstream}
```

If that succeeds, the branch has an upstream and rebasing rewrites history other
people may already have. **Stop and ask** before going any further. Force-pushing
a shared branch is the user's call, and they may prefer fixups at the tip, or a
new branch, or to push nothing at all. If the command fails, there's no upstream
and the history is yours to rewrite.

Group the comments by `commit_sha` and work one target commit at a time, so each
fixup holds only the changes belonging to it:

```
# make only this commit's changes, then:
git add <the files you changed>
git commit --fixup=<commit_sha>
```

Once every group has a fixup commit, squash them all in one non-interactive
rebase:

```
git merge-base <base_sha> <head_sha>          # the rebase target
GIT_SEQUENCE_EDITOR=true git rebase --autosquash <merge-base>
```

`GIT_SEQUENCE_EDITOR=true` is what keeps the rebase from opening an editor.
Without it the rebase hangs waiting for input that will never come.

### Applying commit message edits

Message edits ride along in the same rebase. `git commit --fixup=reword:<sha>`
makes the empty `amend!` commit that autosquash turns into a new message, but it
wants an editor, so feed it one from a file:

```
git commit --fixup=reword:<sha>     # with GIT_EDITOR set to copy your file in
```

The file has to keep git's `amend!` header line, because that header is what
autosquash matches against — overwrite it and the reword lands as a stray commit
of its own instead:

```
amend! <original_subject>

<edited_subject>

<edited_body, hard-wrapped at 72 columns>
```

`original_subject` is in the review JSON alongside the edit, which is what it's
there for. Make these commits before the rebase, next to the fixups, and the one
autosquash pass applies both.

### Comments with no commit_sha

These were authored against the full `base..head` diff and have no single commit
to amend. Make those changes as a new commit at the tip.

### When you're done

Report what you changed, grouped by comment, and note anything you chose not to
do and why. Say explicitly if the branch was rebased, since the user's local
checkout of it is now a different set of SHAs.

## Troubleshooting

### Umpire fails to start under a Bash sandbox

A sandboxed Bash tool blocks both things umpire needs. Starting the server fails
because it can't bind a listener:

```
Error: starting server: listen on 127.0.0.1:0: listen tcp 127.0.0.1:0: bind: operation not permitted
```

And the optional feedback capture after submitting fails because it writes to
`~/.umpire/feedback/`.

Both are fixable with settings, so don't reach for disabling the sandbox. In
`.claude/settings.json` or `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "network": { "allowLocalBinding": true },
    "filesystem": { "write": { "allow": ["~/.umpire"] } }
  }
}
```

Browser auto-open may need more, since `open` goes through LaunchServices and
that wants a Mach lookup the sandbox may not grant. That one only degrades
though: the URL is printed either way, and the user can click it.
