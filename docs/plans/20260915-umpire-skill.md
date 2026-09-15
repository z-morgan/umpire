# The /umpire Skill: Review Round-Trip Inside a Claude Session

## Problem

Umpire is decoupled from Claude by design, and that decoupling costs the user
two context switches per review. Today the loop is: leave the Claude session,
run `umpire` in a terminal, review in the browser, copy the saved review path,
return to Claude, paste it, and explain what to do with it.

Everything needed to close that loop already exists. The review lands at a known
path. The process blocks until the human is finished. The only missing pieces
are a way to start the review from inside a session and a way for the session to
notice when it's done.

## Solution

Ship a Claude Code skill inside the umpire binary. Invoking `/umpire` runs the
same binary the same way — including the browser auto-open — and when the user
submits, the session picks the review up and acts on it without being asked.

The skill is generic. It assumes nothing about the host project's issue tracker,
commit conventions, or the user's personal Claude configuration, because it
installs on every machine that runs `brew install umpire`.

## Mechanism

The Bash tool's `run_in_background` keeps a process alive across turns and
re-invokes the agent when it exits. `umpire` already blocks until the human is
done, so process exit is the signal:

```
/umpire  ->  launch in background  ->  agent ends its turn
             |
             v
    browser opens, user reviews, submits, dismisses
             |
             v
    server shuts down, process exits 0
             |
             v
    harness wakes the agent  ->  read printed path  ->  act on the review
```

No polling, no hooks, no daemon. The discipline the skill has to enforce is
that the agent launches the process and then *stops* — it must not sit in a
wait loop burning the turn.

### Why process exit rather than review submission

`POST /api/review` is not the end of the interaction. The frontend keeps the
server alive to offer feedback recording (`web/static/js/app.js`), and only
calls `/api/shutdown` once the user dismisses that. Exit is therefore a later
and more accurate signal than submission: it means the human has finished with
the UI entirely, not merely that a file was written.

## What the agent does with the review

1. Read the review JSON.
2. If it contains questions, answer them and stop. No code changes until the
   user has replied.
3. Otherwise implement the feedback by **amending the commits the comments
   landed on**, not by appending fixups to the tip.

Each comment carries the `commit_sha` it was authored against, so the mapping
from comment to commit is already in the data. Non-interactively:

```
git commit --fixup=<sha>
GIT_SEQUENCE_EDITOR=true git rebase --autosquash <merge-base>
```

Comments with no `commit_sha` were authored against the full base..head diff and
have no single commit to amend; those become a new commit at the tip.

## Interpreting the review JSON

The anchoring contract currently lives only in Go doc comments on
`review.Comment`, where an agent consuming the JSON will never see it. The skill
has to restate it:

- **`commit_sha` empty** — the comment was authored against the full base..head
  diff, so `line_start` refers to the head revision and can be trusted directly.
- **`commit_sha` set** — `line_start` is relative to *that commit's* diff, and
  `diff_hunk` is the authoritative locator, because a later commit may have
  shifted the same line in the head revision.

Commit message edits carry their own instruction field, and the body is
hard-wrapped at 72 columns per standard git convention.

## Installation

The skill is `go:embed`-ed into the binary and written out by
`umpire install-skill`. It rides along with the existing Homebrew distribution:
no second repo, no marketplace manifest, and `brew upgrade umpire` followed by a
re-run picks up any changes.

Default destination is `~/.claude/skills/umpire/SKILL.md` — user level, because
umpire works in any git repository and a project-scoped install would need
repeating per repo. `--project` writes to `.claude/skills/` instead. An existing
file is not clobbered without `--force`.

Discovery comes from the startup banner, which gains a line pointing at the
command. Like the existing `.gitignore` hint, it is conditional: it prints only
when the skill isn't already installed, so it stops nagging once acted on.

## Output streams

The startup banner writes to stderr. The saved review path — the one line an
agent needs to parse — goes to **stdout**, establishing a split where stdout
carries machine-readable output and stderr carries human narration. Background
task output captures both, so the skill works either way; the split is for
anyone else scripting against umpire.

## Sandbox constraints

Claude Code's Bash sandbox blocks both things umpire needs, verified on 2.1.273:

```
$ python3 -c "socket.bind(('127.0.0.1', 0))"
PermissionError: [Errno 1] Operation not permitted

$ touch ~/.umpire/sandbox-write-test
touch: Operation not permitted
```

The server cannot bind its listener, and the feedback-recording path cannot
write to `~/.umpire/feedback/`. The fix is configuration rather than disabling
the sandbox wholesale:

```json
{
  "sandbox": {
    "network": { "allowLocalBinding": true },
    "filesystem": { "write": { "allow": ["~/.umpire"] } }
  }
}
```

Browser auto-open may additionally require Mach lookup, since `open` goes
through LaunchServices; unverified, and it only degrades to the user clicking
the printed URL.

This affects sandboxed users only, so it belongs in a troubleshooting section
rather than the main flow.

## Implementation Plan

### Step 1: Print the saved review path to stdout

In `handleReview`, after `Store.Save` succeeds, print
`umpire: review saved to <path>` to stdout. The handler currently returns the
path to the browser and nowhere else, leaving the terminal with no record of it.

Files: `internal/server/handlers.go`
Tests: `internal/server/handlers_test.go`

### Step 2: Embed the skill and add `umpire install-skill`

Create `SKILL.md` with the launch half of the flow: pass through any arguments
given to `/umpire`, run the binary in the background, report the URL, end the
turn. Add a cobra subcommand that writes the embedded file, with `--project` and
`--force` flags, printing where it wrote.

Files: `cmd/install_skill.go`, `skills/umpire/SKILL.md`, `cmd/root.go`
Tests: `cmd/install_skill_test.go`

### Step 3: Advertise the command in the startup banner

Add a line to the banner pointing at `umpire install-skill`, conditional on the
skill not already being present, alongside the existing `.gitignore` hint.

Files: `cmd/root.go`

### Step 4: Add the pickup half to the skill

Extend `SKILL.md`: read the background task output, find the printed path, fall
back to the newest file in `.umpire/reviews/` if it isn't there. Then the
interpretation rules above, the questions-first rule, and the amend-the-right-
commit flow.

Files: `skills/umpire/SKILL.md`

### Step 5: Guard rails

The one that matters: check whether the branch has already been pushed before
rewriting it. A successful `git rev-parse @{upstream}` means a rebase rewrites
published history, which is the user's call and not the agent's. Also: the
cancel path, base and head resolving to the same commit, and the sandbox
troubleshooting section.

Files: `skills/umpire/SKILL.md`

### Step 6: Documentation

Document `install-skill` and the `/umpire` flow in the README. Add the sandbox
keys to this repo's `.claude/settings.local.json`.

Files: `README.md`, `.claude/settings.local.json`

## Design Decisions

**Why a skill rather than a slash command?** Both are invoked as `/umpire`. A
command is a prompt template and would be sufficient for explicit invocation
alone, but a skill can also be selected by the model — "review this before we
merge" reaches it without the user remembering the name — and it supports
bundled resources if the instructions later outgrow one file.

**Why `go:embed` rather than a plugin marketplace?** A marketplace listing is
more discoverable and updates through `/plugin`, but it means a second repo and
a manifest to keep in sync with the brew tap. Embedding means the skill and the
binary that it drives are the same artifact and cannot drift apart.

**Why not write the skill automatically on first run?** Zero friction, but
writing into someone's home configuration without being asked is the kind of
surprise that makes people distrust a tool. A printed hint costs one line.

**Why amend commits instead of appending fixups?** A review comment is about the
commit that introduced the line. Appending a fixup leaves the branch reading as
a sequence of mistakes and corrections; amending leaves it reading as though the
feedback had been incorporated the first time. The data already supports this —
every comment knows its commit.

**Why stop on questions rather than guessing?** Review comments are not
uniformly change orders. Some are questions, some are disagreements. An agent
that treats "why did you do it this way?" as an instruction produces a confident
non-answer and edits code that didn't need editing.

**Why does the skill stay generic?** It installs on every machine that runs
`brew install umpire`. Baking in one user's issue tracker or commit rhythm would
make it worse for everyone else and would duplicate guidance their own
configuration already provides.
