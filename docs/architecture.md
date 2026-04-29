# Architecture

skill-sync uses a **mesh/P2P sync model**: there is no designated source of truth. Every configured target directory is a peer. The skill with the strongest claim wins and propagates to all others.

---

## Component Overview

```
Config (skill-sync.yaml)
        │
        ▼
    Scanner          finds all SKILL.md files across all target dirs
        │
        ▼
    Resolver         picks the winner per skill name
        │
        ▼
     Engine          writes the winner to every target that needs updating
```

---

## Scanner (`internal/mesh/scanner.go`)

Walks each target directory recursively looking for files named exactly `SKILL.md`. For every file found:

- Derives the **skill ID** from the parent directory name (e.g., `~/.claude/skills/my-skill/SKILL.md` → `my-skill`)
- Computes a **SHA-256 hash** of the file content
- Parses the **YAML frontmatter** to extract `version` and `name`
- Records the file's **modification time** (`mtime`)

Returns a map of `skillID → []SkillInstance` where each instance represents one copy of that skill across the targets.

If a target directory does not exist, it is silently skipped — not an error.

---

## Resolver (`internal/mesh/resolver.go`)

Given a list of instances for the same skill, picks the winner using this precedence:

| Priority | Rule |
|----------|------|
| 1st | All hashes identical → no-op, any instance is returned |
| 2nd | Higher `version` in frontmatter wins |
| 3rd | Same version → newer `mtime` wins |
| 4th | Same version + same mtime + different hash → **conflict**, skill is skipped |

`version` is parsed as `float64`, so `1.0`, `1.1`, and `2` all compare correctly. Skills with no frontmatter or no version field get `0.0`.

Conflicts are never silently resolved — they are surfaced to the user and the skill is left untouched.

---

## Engine (`internal/sync/engine.go`)

Takes the winning `SkillInstance` and propagates it to every target directory:

1. Reads the winner file content
2. For each target dir, skips it silently if its **parent** directory doesn't exist (avoids creating directories for tools that aren't installed)
3. Creates the destination skill subdirectory if needed
4. Calls `AtomicWrite` to write the file

---

## Atomic Writer (`internal/writer/atomic.go`)

Every write is atomic to avoid partial files on crash or power loss:

1. If the destination file already exists, copy it to `<dest>.bak`
2. Write content to a temp file in the same directory (`atomic-tmp-*`)
3. `fsync` the temp file to flush OS buffers to disk
4. `os.Rename` the temp file over the destination (atomic on all POSIX systems; best-effort on Windows)

The `.bak` file is always the last known good version before an overwrite.

---

## Config (`internal/agent/registry.go`)

The config file (`skill-sync.yaml`) uses a flat list of target paths:

```yaml
targets:
  - ~/.claude/skills
  - ~/.cursor/skills
  - ~/.gemini/skills
```

Tilde (`~`) is expanded to the user's home directory at parse time. Paths are not validated at config load — missing directories are handled gracefully by the Scanner and Engine at runtime.

---

## TUI (`internal/tui/`)

Running `skill-sync` with no arguments launches a [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI. Views:

| View | File | Purpose |
|------|------|---------|
| Menu | `tui/menu/` | Main navigation |
| Sync Select | `tui/sync_select_view/` | Choose which skills to sync |
| Config | `tui/config_view/` | Inspect and edit targets |
| Init | `tui/init_view/` | Bootstrap a new config file |
| Output | `tui/output_view/` | Scrollable sync/verify results |

The TUI and CLI commands (`sync`, `verify`) share the same underlying `Scanner → Resolver → Engine` pipeline.
