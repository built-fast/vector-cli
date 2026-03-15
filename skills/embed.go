package skills

import "embed"

// Content embeds the skills directory tree (e.g., vector/SKILL.md).
//
//go:embed vector
var Content embed.FS
