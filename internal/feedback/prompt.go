package feedback

import "fmt"

// Threshold is the minimum number of snapshots before analysis is offered.
const Threshold = 5

// GeneratePrompt builds a self-contained prompt the user can paste into a
// Claude Code session to analyze accumulated feedback and propose config updates.
func GeneratePrompt(count int) string {
	return fmt.Sprintf(`I have %d code-review feedback snapshots saved in ~/.umpire/feedback/ that I'd like you to analyze. Each snapshot is a JSON file containing a diff I reviewed plus my review comments. Please do the following:

1. Read all snapshot-*.json files in ~/.umpire/feedback/.
2. For each snapshot, study the diff and my review comments to understand what I was correcting or requesting.
3. Classify each piece of feedback as either a **generalizable preference** (naming conventions, error handling style, architectural patterns, testing expectations, code organization) or a **one-off correction** (typo, specific bug fix, context-dependent change). Only generalizable preferences should produce config changes.
4. Look for recurring themes across the snapshots. Group related feedback into categories.
5. Read my existing user-level Claude configuration:
   - ~/.claude/CLAUDE.md (global instructions)
   - ~/.claude/settings.json (global settings)
6. Propose specific additions or edits to those files that would address the patterns you identified. Do not duplicate or contradict existing rules. Only modify user-level config — do not touch project-level files.
7. Present each proposed change for my approval before applying it. Show what you'd add or modify, and explain which feedback snapshots motivated the change.
8. After I approve and you apply the changes, delete the processed snapshot files from ~/.umpire/feedback/.`, count)
}
