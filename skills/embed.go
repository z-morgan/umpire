package skills

import _ "embed"

// UmpireSKILL is the /umpire Claude Code skill, embedded so that it ships with
// the binary and cannot drift from the version of umpire it drives.
//
//go:embed umpire/SKILL.md
var UmpireSKILL string
