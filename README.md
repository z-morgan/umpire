# umpire

Local code review tool. Umpire gives you a GitHub-like review UI in the browser for reviewing feature branch commits before pushing.

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
- Inline commenting on any diff line
- Review summary with submit
- Dark mode (toggle with `d`, or follows system preference)
- Keyboard shortcuts: `j`/`k` to navigate files, `n`/`p` for commits
- Reviews saved as JSON for scripting and CI integration
