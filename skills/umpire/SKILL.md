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

**This is the first thing you do. Not the second.**

One Bash call, `run_in_background: true`, sandbox off:

```
umpire
```

Pass any arguments given to `/umpire` straight through. `umpire --base develop`
and `umpire --head feature/auth` are the common ones; `umpire --help` lists the
rest.

Run nothing before it. Not `git status`, not `git log`, not a check for whether
the branch has commits worth reviewing, not a look at what the base ought to be.
Umpire resolves the refs itself and prints what it found. The user typed
`/umpire` expecting a browser window, and every command you run first is a
second they spend watching a spinner instead. Getting the review open is the
whole job.

If something turns out to be off — no commits between base and head, a base that
isn't what they meant — umpire's banner says so, and you can raise it *after*
it's running. Don't front-load it.

### The sandbox has to be off for this one call

Umpire binds a port on 127.0.0.1 and opens a browser window. A Bash sandbox
blocks both, and the second one can't be allowlisted around: the browser handoff
is refused at the Mach layer, so `open` activates the browser without delivering
the URL and the user gets a focused window with no new tab. Launching the
browser binary directly fails the same way.

So if your Bash tool sandboxes commands, run umpire outside the sandbox. This is
a local review UI on the loopback interface that the user asked for by name.

If you can't disable it, umpire still starts as long as port binding is
permitted, and still prints its URL — say the auto-open didn't work and hand
them the URL to open.

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

Keep the message short: the review is open, and you'll pick it up when they're
done. Include the URL if the launch result already has it.

Say that submitting isn't the last step. After they submit, umpire asks whether
to record the feedback, and it keeps running until they answer — that answer is
what closes the server and hands the session back to you. A user who submits and
walks away has done the obvious thing and will be left waiting on a pickup that
can't happen, so it's worth the one sentence up front.

Umpire prints a line when it couldn't open the browser. If you see that line,
say so and hand over the URL — a failed auto-open can leave the browser focused
with no new tab, which looks enough like success that the user will sit waiting
on a review that never loaded. If you don't see it, the browser opened.

Either way, don't go back to the output for the URL a second time. That's the
wait loop above, wearing a different hat.

### If the user changes their mind

A review that is never submitted never exits, so the background task would sit
there indefinitely. If the user says to cancel, forget it, or moves on to
something else, kill the task. Don't leave it running on the chance they come
back to it.

## Finding the review

The harness wakes you with the process's output when umpire exits, which happens
once the user answers the feedback question that follows their submission — not
at submission itself. Umpire prints the path it saved the review to:

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

Running umpire outside the sandbox fixes both, and is the recommended answer
because it's also the only thing that fixes the browser. If you'd rather keep
umpire sandboxed and open the URL by hand, these two keys get it far enough to
serve the review, in `.claude/settings.json` or `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "network": { "allowLocalBinding": true },
    "filesystem": { "write": { "allow": ["~/.umpire"] } }
  }
}
```

Settings changes land at session start, so a session already running won't see
them.

### The browser focuses but no tab opens

Auto-open fails under a sandbox and those two keys don't fix it. LaunchServices
activates the browser but refuses to deliver the Apple Event carrying the URL:

```
couldn't open a browser automatically, so open the URL above yourself
(exit status 1: ... Code=-600 "procNotFound" ... _LSFunction=_LSAnnotateAndSendAppleEventWithOptions)
```

The browser coming to the front is the app being activated. The missing tab is
the refused event.

There's no allowlist entry for this. Launching the browser binary directly
instead of going through `open` fails too, on the same Mach restriction. The
only fix is running umpire outside the sandbox. Short of that, umpire still
serves the review at the printed URL and everything after that works normally,
so the user opening that URL by hand costs them one click.
