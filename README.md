# umpire

Umpire is a local code review UI that aggregates your feedback to give back to your AI coding agent. It also tracks your feedback across sessions so you can teach your agent to align with your preferences over time.

## Install

```
brew install z-morgan/tap/umpire
```

Or with Go:

```
go install github.com/zmorgan/umpire@latest
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

## Features

- Commit-by-commit or full-diff view with syntax-highlighted diffs
- Expand additional context lines around any hunk
- Inline commenting on any diff line
- Propose commit message rewrites (subject and body) alongside your review
- Resizable sidebar and commit-message panes
- Review summary with submit, saved as JSON for scripting and CI integration
- Optional feedback capture after submitting: record your reviews and generate a prompt for Claude to propose config improvements
- Keyboard shortcuts: `j`/`k` to navigate files, `←`/`→` to move between commits
