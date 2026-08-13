# Agent Instructions

This project uses **bd** (beads) for issue tracking. Run `bd onboard` to get started.

## Quick Reference

```bash
bd ready              # Find available work
bd show <id>          # View issue details
bd update <id> --status in_progress  # Claim work
bd close <id>         # Complete work
bd sync               # Sync with git
```

## Wrapping up a work session

When you're ending a session, it helps to leave the repo tidy for whoever
picks it up next:

1. **File issues for remaining work** - Create issues for anything that needs follow-up.
2. **Run quality gates** (if code changed) - Tests, linters, builds.
3. **Update issue status** - Close finished work, update in-progress items.
4. **Commit locally** - Commit your work and run `bd sync` to record the issue ledger.
5. **Hand off** - Leave context for the next session.

Umpire is a shared repo, so treat this file as guidance rather than
orders. In particular, leave the push to the human: commit locally and
let the maintainer review and push when they're ready. If you clone this
repo and your own agent reads this file, that's the norm we'd suggest,
not one we're imposing.

