# umpire

Umpire is a local code review UI that aggregates your feedback to give back to your AI coding agent. It also tracks your feedback across sessions so you can teach your agent to align with your preferences over time.

## Install

```
brew install z-morgan/tap/umpire
```

## Usage

Run from any git repository:

```
umpire
```

This diffs the current branch against `main`, opens a browser with the review UI, and waits for you to submit your review. Reviews are saved as JSON files in `.umpire/reviews/`.

### Flags

```
--base string   base branch to diff against (default "main")
--head string   head ref to review (default: current branch)
--port int      port to serve on (default: auto)
```

### Examples

Review the current branch against main:

```
umpire
```

Review against a different base branch:

```
umpire --base develop
```

Review a specific branch:

```
umpire --base main --head feature/auth
```

## Reviewing from a Claude Code session

Umpire ships a Claude Code skill that closes the loop, so you don't have to
leave your session to run a review and then come back and explain the result.
Install it once:

```
umpire install-skill
```

That writes `~/.claude/skills/umpire/SKILL.md`, which covers every repository on
the machine. Use `--project` to scope it to the current repo instead, and
`--force` to overwrite an existing file. The skill is embedded in the binary, so
`brew upgrade umpire` plus a re-run picks up any changes to it.

Then, in a session:

```
/umpire
```

Claude starts umpire in the background and ends its turn while you review. When
you submit and dismiss the UI, the process exits, and Claude picks the review up
and works it: answering any questions you asked first, and otherwise amending
the commits your comments landed on rather than piling fixups at the tip.

Arguments pass straight through, so `/umpire --base develop` works the way the
CLI flag does.

### Under a Bash sandbox

A sandboxed Bash tool blocks umpire from binding its listener and from writing
feedback to `~/.umpire/`. Two settings keys fix it, in `.claude/settings.json`
or `.claude/settings.local.json`:

```json
{
  "sandbox": {
    "network": { "allowLocalBinding": true },
    "filesystem": { "write": { "allow": ["~/.umpire"] } }
  }
}
```

## Features

- Commit-by-commit or full-diff view with syntax-highlighted diffs
- Expand additional context lines around any hunk
- Inline commenting on any diff line
- Propose commit message rewrites (subject and body) alongside your review
- Resizable sidebar and commit-message panes
- Review summary with submit, saved as JSON for scripting and CI integration
- Optional feedback capture after submitting: record your reviews and generate a prompt for Claude to propose config improvements
- Keyboard shortcuts: `j`/`k` to navigate files, `←`/`→` to move between commits
- A `/umpire` skill for Claude Code, so a review round-trips without leaving the session
