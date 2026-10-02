# Spec: `skill-import`

**Change ID:** skill-import  
**Branch:** feat/import_skills  
**Date:** 2026-06-05  
**Status:** APPROVED

---

## 1. Overview

`skill-import` adds a `skill-sync import <skill-name>` CLI sub-command and two
new TUI screens that allow a user to copy one skill directory from any
globally-configured agent source into the local project at
`./.agents/skills/<skillID>/`.

The operation is a **one-shot snapshot**. Imported skills are not tracked by
`sync` or `verify`. The existing mesh pipeline is unchanged.

---

## 2. Functional Requirements

### FR-1 Source discovery

- Sources are the `targets:` list in `skill-sync.yaml`, resolved via
  `internal/agent/config_resolver` and `internal/agent/registry` (tilde
  expansion included).
- Each target's `skills/` subdirectory is walked; any directory that directly
  contains a `SKILL.md` file is treated as a skill.
- If `skill-sync.yaml` is absent, the command exits with a descriptive error:
  `"no skill-sync.yaml found; run 'skill-sync init' to create one"`.

### FR-2 CLI sub-command

```
skill-sync import <skill-name>
```

- Exactly **one positional argument** (the skill ID / directory name).
- No `--all`, no `--target`, no `--dry-run` (all deferred to v2).
- Config file flag `-c / --config` inherited from root command.
- Exit codes:
  - `0` — success (skill copied or already up-to-date).
  - `1` — any error (not found, config missing, copy failure).
- Stdout: single-line confirmation `"Imported <skillID> from <sourcePath>"`.
- Stderr: errors only.

### FR-3 Destination

- Always `./.agents/skills/<skillID>/` relative to the process working
  directory.
- Created with `os.MkdirAll` when absent.
- Destination root `./.agents/skills/` is **never** registered in
  `skill-sync.yaml` automatically. The user controls their config.

### FR-4 Conflict resolution

| Situation | CLI behaviour | TUI behaviour |
|---|---|---|
| Skill found in one source | Import directly | Import directly |
| Skill found in multiple sources, clear winner (version/mtime rules) | Auto-pick winner, silent | Auto-pick winner, show source in output_view |
| Skill found in multiple sources, true tie (same version + mtime + different hash) | Exit 1, message: `"conflict: <skillID> has identical version and mtime in multiple sources; resolve manually"` | Show tie-break choice screen (future — deferred; exit 1 for v1) |
| Skill not found in any source | Exit 1, message: `"skill '<skillID>' not found in any configured source"` | Show error in output_view |

> **v1 decision:** True tie resolution in TUI is deferred. Both CLI and TUI
> exit with an error on a true tie. A future task will add the interactive
> tie-break choice.

### FR-5 Idempotency

Running `import <skillID>` twice with an unchanged source is a no-op: the
existing directory is overwritten but the result is byte-identical. Exit `0`.

### FR-6 Backup on overwrite / Transactional Copy

To guarantee that the destination directory is never left in a partially-written or corrupt state if an import fails:

1. Copy all new skill files into a temporary directory `./.agents/skills/<skillID>.tmp/` (creating it and nested directories as needed) writing each file atomically via `writer.AtomicWrite`.
2. If any write or copy operation fails:
   - Clean up the temporary directory `./.agents/skills/<skillID>.tmp/` via `os.RemoveAll`.
   - Return the error. The existing `./.agents/skills/<skillID>/` (if any) remains completely untouched.
3. If all files are copied successfully:
   - Remove any stale backup directory `./.agents/skills/<skillID>.bak/` if present (`os.RemoveAll`).
   - If `./.agents/skills/<skillID>/` already exists, rename it to `./.agents/skills/<skillID>.bak/` (`os.Rename`).
   - Rename `./.agents/skills/<skillID>.tmp/` to `./.agents/skills/<skillID>/` (`os.Rename`).

### FR-7 TUI screens

Two new screens wired into `internal/tui/program.go`:

**Screen A — import_source_view**

- Lists all agent directories from `cfg.ExpandTargets(home)`.
- Each row shows the agent short name (last path component) and its full path.
- Per-row toggle (reuse `shared_toggle`) to exclude sources. All start checked.
- Session-only: toggles are not persisted to `skill-sync.yaml`.
- Confirm key (`Enter` / `ctrl+s`): transitions to Screen B with the filtered
  source list.
- Cancel key (`ctrl+c` / `q`): returns to main menu.

**Screen B — import_skill_view**

- Lists all skills found across the checked sources.
- Each row: `<skillID>` | `<name from YAML frontmatter or skillID>` |
  `<description from YAML frontmatter, truncated to 60 chars>`.
- Single-selection (no multi-select in v1).
- Confirm key (`Enter`): runs import for the selected skill → transitions to
  existing `output_view`.
- Cancel key (`esc`): returns to Screen A.
- If no skills are found (all sources empty), show inline message: `"No skills
  found in the selected sources."` with only a Back option.

**Menu entry**

- Add `"Import"` item to `internal/tui/menu/model.go`, transitioning to Screen A.

---

## 3. Package API: `internal/importer`

### 3.1 Types

```go
// SkillCandidate is a skill found during source discovery.
type SkillCandidate struct {
    SkillID   string          // directory name (e.g. "git-expert")
    SourceDir string          // absolute path to the skill directory
    AgentDir  string          // absolute path to the parent target
    Metadata  models.SkillMetadata
    Mtime     time.Time
    Hash      string          // SHA-256 of directory manifest (see §3.4)
}

// Importer holds resolved configuration and drives import operations.
type Importer struct {
    Targets []string // resolved, expanded target directories
    DestRoot string  // absolute path to ./.agents/skills/
}
```

### 3.2 Constructor

```go
// New returns an Importer for the given expanded targets and dest root.
// targets must already be tilde-expanded absolute paths.
// destRoot is typically filepath.Join(cwd, ".agents", "skills").
func New(targets []string, destRoot string) *Importer
```

### 3.3 FindSkill

```go
// FindSkill walks each target's skills/ subdirectory and returns all
// candidates whose directory name matches skillID (case-sensitive).
// Returns an empty slice (not an error) when no match is found.
func (imp *Importer) FindSkill(skillID string) ([]SkillCandidate, error)
```

- Walks `<target>/skills/` for each target.
- A directory is a skill candidate if it contains a `SKILL.md` directly
  (not recursively).
- YAML frontmatter (`name`, `description`, `version`) is parsed from
  `SKILL.md`; missing fields default to zero values (version `0`, name `""`).
- Hash computed via `manifest.DirHash(skillDir)` (see §3.4).
- `os.Stat` provides mtime from the `SKILL.md` file (not the directory).

### 3.4 manifest.DirHash

```go
// DirHash computes a deterministic, cross-platform SHA-256 over all files in dir.
// Algorithm:
//   1. Walk dir recursively, collect relative paths of all regular files.
//   2. Normalize all relative paths to use forward slashes (e.g. filepath.ToSlash(relPath))
//      to ensure cross-platform hash determinism.
//   3. Sort normalized relative paths lexicographically.
//   4. For each path: hash "<normalizedRelPath>\n<fileContentBytes>".
//   5. Final hash: SHA-256 of the concatenation of all per-file hashes.
func DirHash(dir string) (string, error)
```

Returned as a hex string (64 chars). Symlinks are skipped.

### 3.5 Resolve

```go
// Resolve converts candidates to SkillInstances and delegates to
// mesh.NewDefaultResolver().Resolve(). Returns the winning candidate,
// a conflict flag, and any error.
func Resolve(candidates []SkillCandidate) (*SkillCandidate, bool, error)
```

Conversion: `SkillCandidate → models.SkillInstance` sets:
- `ID` = `SkillCandidate.SkillID`
- `Path` = `filepath.Join(SkillCandidate.SourceDir, "SKILL.md")`
- `TargetDir` = `SkillCandidate.AgentDir`
- `Hash`, `Mtime`, `Metadata` copied directly.

### 3.6 CopySkill

```go
// CopySkill copies the full skill directory from src into
// filepath.Join(destRoot, skillID), applying the transactional backup strategy (§FR-6).
// src is the absolute path to the source skill directory.
func (imp *Importer) CopySkill(skillID, src string) error
```

- `os.MkdirAll(destRoot, 0o755)` to ensure the destination root exists.
- Construct the temporary path: `tmpDir := filepath.Join(destRoot, skillID + ".tmp")`.
- Ensure clean up of `tmpDir` on failure (e.g. using a deferred function or clean up on error return).
- Walk `src` recursively; for each regular file:
  - Compute relative path `relPath` to `src`.
  - Compute temporary dest path = `filepath.Join(tmpDir, relPath)`.
  - `os.MkdirAll` for intermediate directories under `tmpDir`.
  - Read file content.
  - Call `writer.AtomicWrite(tmpDestPath, content)`.
- Symlinks are skipped.
- If walking and copying succeeds:
  - Remove stale backup: `os.RemoveAll(filepath.Join(destRoot, skillID + ".bak"))`.
  - If target `filepath.Join(destRoot, skillID)` exists, rename it to `filepath.Join(destRoot, skillID + ".bak")`.
  - Rename `tmpDir` to the final target path `filepath.Join(destRoot, skillID)`.

---

## 4. Data Models (additions to `internal/models`)

No new exported types in `models` package. `SkillCandidate` lives in
`internal/importer` to avoid circular imports.

`SkillMetadata` already has `Version` and `Name`. Add `Description`:

```go
type SkillMetadata struct {
    Version     float64 `yaml:"version"`
    Name        string  `yaml:"name"`
    Description string  `yaml:"description"` // NEW
}
```

---

## 5. Error Taxonomy

| Sentinel / package | Condition | User message |
|---|---|---|
| `cli.ErrConfigNotFound` (`internal/cli`) | `skill-sync.yaml` not found by resolver | `"no skill-sync.yaml found; run 'skill-sync init' to create one"` |
| `importer.ErrSkillNotFound` (`internal/importer`) | `FindSkill` returns empty candidates | `"skill '<id>' not found in any configured source"` |
| `importer.ErrConflict` (`internal/importer`) | `Resolve` returns `conflicts=true` | `"conflict: '<id>' has identical version and mtime in multiple sources; resolve manually"` |
| wrapped `fs` errors | Copy, backup, or mkdir failures | error message from `fmt.Errorf("...: %w", err)` |

`ErrConfigNotFound` is defined in `internal/cli` because config resolution
happens at the CLI layer before `Importer` is constructed. `ErrSkillNotFound`
and `ErrConflict` are exported sentinels in `internal/importer`. All three are
checked with `errors.Is`.

---

## 6. CLI Command (`internal/cli/import.go`)

```go
func NewImportCmd() *cobra.Command
```

Flow:

1. Validate exactly one positional arg; return usage error otherwise.
2. Resolve config via `config_resolver.New().Resolve(cfgFlag)`.
3. Parse config with `agent.ParseConfig`; expand targets with
   `cfg.ExpandTargets(home)`.
4. Construct `importer.New(targets, destRoot)`.
5. Call `FindSkill(skillID)` → handle `ErrSkillNotFound`.
6. Call `Resolve(candidates)` → handle `ErrConflict`.
7. Call `CopySkill(skillID, winner.SourceDir)`.
8. Print `"Imported <skillID> from <winner.SourceDir>"` to stdout.

No interactive output, no spinner. Suitable for scripting.

---

## 7. TUI Integration (`internal/tui/program.go`)

### New state constants

```go
stateImportSource  // Screen A
stateImportSkill   // Screen B
```

### New message types

```go
type importSourceConfirmedMsg struct{ sources []string }
type importSkillSelectedMsg   struct{ candidate importer.SkillCandidate }
type importSkillBackMsg        struct{}
```

### State machine additions

```
stateMenu
  → (user selects "Import") → stateImportSource
stateImportSource
  → (confirm) → stateImportSkill
  → (cancel)  → stateMenu
stateImportSkill
  → (confirm) → run import → stateOutput
  → (back)    → stateImportSource
stateOutput
  → (quit/back) → stateMenu
```

`stateOutput` is the existing `output_view`.

---

## 8. Test Requirements

### 8.1 `internal/importer`

| Test | Type | Notes |
|---|---|---|
| `FindSkill` returns candidates from multiple targets | unit | `t.TempDir()`, populate skills/ dirs |
| `FindSkill` returns empty for unknown skill ID | unit | |
| `FindSkill` skips targets whose skills/ dir does not exist | unit | |
| `FindSkill` reads YAML frontmatter (name, description, version) | unit | |
| `DirHash` is deterministic and order-independent | unit | two dirs with same files in different creation order |
| `DirHash` changes when a file changes | unit | |
| `Resolve` picks highest version winner | unit | delegate to mesh resolver rules |
| `Resolve` picks newest mtime when versions match | unit | |
| `Resolve` returns conflict=true on tie | unit | |
| `CopySkill` copies full directory tree | integration | real temp dirs |
| `CopySkill` creates backup when dest exists | integration | |
| `CopySkill` removes stale .bak before backup | integration | |
| `CopySkill` is idempotent on identical source | integration | |
| `CopySkill` returns error and leaves existing target untouched when write fails | integration | inject read-only file |

### 8.2 `internal/cli/import.go`

| Test | Type | Notes |
|---|---|---|
| Happy path: imports skill, exits 0, prints confirmation | integration | real temp dirs + real SKILL.md |
| Missing config file: exits 1 with clear message | integration | |
| Skill not found: exits 1 with clear message | integration | |
| Wrong arg count (0 or 2+): exits 1 with usage error | unit | |
| Conflict: exits 1 with conflict message | integration | two sources with identical version/mtime/different hash |

### 8.3 TUI views

| View | Required golden files |
|---|---|
| `import_source_view` | `initial.txt`, `one_source_unchecked.txt`, `all_unchecked.txt` |
| `import_skill_view` | `initial.txt`, `skill_selected.txt`, `no_skills.txt` |

All TUI tests use `teatest` + `t.TempDir()`.

### 8.4 `internal/models`

- Add a test case to the existing `SkillMetadata` unmarshal test verifying
  that `description` round-trips correctly.

---

## 9. Non-Functional Requirements

- No new external Go module dependencies.
- `go test ./...` must pass on Linux with zero failures.
- Windows: `CopySkill` backup rename may fail on cross-device moves; guard
  with `t.Skip` on Windows where necessary.
- `go vet ./...` and `staticcheck ./...` clean.
- All new exported symbols have GoDoc comments.

---

## 10. Out of Scope (confirmed)

- Multi-destination import
- `--all` flag
- `--target` flag
- `--dry-run` flag
- Persisting agent-filter choices to `skill-sync.yaml`
- Unimport / remove command
- Interactive tie-break in TUI (deferred to v2)
- Mesh-sync directory-copy bug fix (tracked separately)
- Any change to `sync`, `verify`, scanner, engine, or mesh resolver
