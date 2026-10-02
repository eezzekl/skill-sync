# Proposal: `skill-import`

**Change ID:** skill-import  
**Branch:** feat/import_skills  
**Date:** 2026-05-26  
**Status:** DRAFT

---

## Intent

Add a one-shot `import` flow — available as both a CLI sub-command and a
new TUI screen sequence — that lets a user selectively copy ONE skill
(its full directory tree) from any globally configured agent source into
the local project at `./.agents/skills/<skillID>/`.

This is a *snapshot* operation, not mesh synchronisation. Imported skills
are invisible to `sync` and `verify`.

> **Reusable primitive note:** this change introduces
> `internal/importer/` — a new package that implements correct
> full-directory copy (temp-dir → rename, per-file atomic writes,
> `.bak` backup). That primitive will later be reused by the mesh-sync
> directory-copy bug fix tracked in `context/pending-fixes.md`.
> The mesh fix is **out of scope here**.

---

## Motivation

The existing mesh sync propagates skills between peer agent directories
that are all configured in `skill-sync.yaml`. There is no mechanism to
pull a single skill from the global pool into a project's own skill
directory. Users currently copy skill directories manually, which is
error-prone and bypasses the version / conflict logic already baked into
`internal/mesh/resolver.go`.

---

## Scope

### In scope (this change)

- `skill-sync import <skill-name>` CLI sub-command (exactly one skill
  per invocation, no variadic args, no `--all`, no `--target` flag).
- TUI: two new screens wired into `internal/tui/program.go`:
  - **Import-source screen** — lists detected agents from `targets:` with
    per-row toggle to exclude any. Toggles are session-only (not
    persisted).
  - **Skill-select screen** — lists skills found across the included
    sources; each row shows `name` and `description` from YAML
    frontmatter (fallback: directory name / empty string).
  - User picks one → import runs → existing `output_view`.
- New `internal/importer/` package:
  - `Importer` type that holds the target list + destination root.
  - `CopySkill(skillID, sourceDir, destRoot string) error` — directory
    copy primitive (temp + fsync + rename per-file; see Backup Strategy
    below).
  - `FindSkill(name string, targets []string) ([]SkillCandidate, error)`
    — locates all matching skill directories across sources.
  - `Resolve(candidates []SkillCandidate) (*SkillCandidate, bool, error)`
    — thin wrapper that delegates to `mesh.Resolver` winner rules.
- Destination: `./.agents/skills/<skillID>/` relative to `cwd`.
  Created with `os.MkdirAll` when absent.
- Source resolution via existing `internal/agent/config_resolver` +
  `internal/agent/registry` (tilde expansion included). Missing config
  → user-facing error suggesting `skill-sync init`.
- Conflict when the same `skillID` exists in multiple sources:
  - CLI mode: auto-pick winner (idempotent, scriptable).
  - TUI mode: present conflict to user as a tie-break choice.
- Strict TDD for all new packages (`t.TempDir()`, table-driven,
  `teatest` + golden files for TUI screens).

### Out of scope (explicitly deferred)

- Multi-destination (`.claude/skills`, `.cursor/skills`, etc.)
- `--all` flag or variadic skill arguments
- `--target` flag
- Persisting the agent-filter selection to `skill-sync.yaml`
- Removing / unimporting skills
- Dry-run mode
- Mesh-sync directory-copy bug fix (tracked in `context/pending-fixes.md`)
- Any change to `sync`, `verify`, or their tests

---

## Approach

### Source Discovery

Re-use `internal/agent/config_resolver.New().Resolve("")` to locate
`skill-sync.yaml`. Parse it with `agent.ParseConfig` and call
`cfg.ExpandTargets(home)` to get the resolved list of target parent
directories. Each target's `skills/` subdirectory is then walked for
skill directories (any directory containing a `SKILL.md`).

### Conflict Resolution

`internal/importer.Resolve` converts each candidate into a
`models.SkillInstance` (by reading `SKILL.md` frontmatter for version,
using `os.Stat` for mtime, and computing a SHA-256 of the
directory manifest — sorted relative paths + per-file hashes) and
delegates to `mesh.NewDefaultResolver().Resolve(instances)`.

> Hash computation uses the *directory manifest* rather than a single
> file, because the mesh scanner's `SKILL.md`-only hash is the known
> pending bug. This keeps the import correct without touching the
> existing resolver logic.

### Backup Strategy

When `./.agents/skills/<skillID>/` already exists:

1. Remove any stale `.bak` directory: `os.RemoveAll(<skillID>.bak/)`.
2. Rename existing directory to `<skillID>.bak/`.
3. Copy new skill files one-by-one via `writer.AtomicWrite` (temp +
   fsync + rename) into a fresh `<skillID>/` directory.

Step 2 is a single `os.Rename` — on POSIX this is atomic at the
directory level. Step 3 uses the existing `writer` contract per file.
If any file write in step 3 fails, the backup at step 2 preserves the
previous state and the caller returns an error.

### New Package Layout

```
internal/
  importer/
    importer.go          — Importer type, FindSkill, CopySkill, Resolve
    manifest.go          — directory manifest + SHA-256 helper
    importer_test.go     — table-driven unit + integration tests
    manifest_test.go
internal/
  tui/
    import_source_view/
      model.go
      model_test.go
      testdata/golden/   — teatest golden snapshots
    import_skill_view/
      model.go
      model_test.go
      testdata/golden/
internal/
  cli/
    import.go            — `skill-sync import <skill-name>` Cobra command
    import_test.go
```

`cmd/skill-sync/main.go` and `internal/cli/root.go` gain one
`cmd.AddCommand(NewImportCmd())` call.

`internal/tui/program.go` gains two new `state` constants
(`stateImportSource`, `stateImportSkill`), message types for each view
transition, and the corresponding `Update`/`View` branches.

---

## Affected Areas

| Area | Change |
|---|---|
| `internal/importer/` | **New package** — core import logic |
| `internal/cli/import.go` | **New file** — Cobra command |
| `internal/cli/root.go` | +1 `AddCommand` call |
| `internal/tui/program.go` | +2 states, +2 message types, update/view branches |
| `internal/tui/import_source_view/` | **New view** |
| `internal/tui/import_skill_view/` | **New view** |
| `internal/tui/menu/` | +1 menu item ("Import") |
| `cmd/skill-sync/main.go` | No change expected (root already delegates to `cli`) |
| `internal/mesh/` | No change |
| `internal/sync/` | No change |
| `internal/writer/` | No change — reused as-is |
| `verify` command | No change — imported skills are not tracked |

---

## Risks

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Directory rename in backup step not atomic on all OSes (e.g. cross-device) | Low | Medium | `os.Rename` is in-place on POSIX; document Windows caveat, add `t.Skip` for cross-device tests |
| `os.RemoveAll` on `.bak/` deletes the previous backup permanently | Low | Low | Documented limitation; future `--keep-bak` flag is deferred |
| Conflict auto-resolution in CLI mode overwrites a manually-edited local skill | Medium | Medium | Backup-before-overwrite (step 2) provides one-generation recovery |
| Walking `skills/` across all targets is slow if targets are remote/mounted filesystems | Low | Low | v1 does not address network targets; same limitation applies to existing `sync` |
| New TUI states increase `program.go` complexity | Medium | Low | Extracted into dedicated view packages; `program.go` changes are mechanical state-machine additions |

---

## Rollback Plan

1. `git revert` the commit(s) in `feat/import_skills` — no data
   migration needed.
2. The `./.agents/skills/` directory is project-local and not tracked
   by `skill-sync.yaml`, so reverting the binary does not break
   existing mesh-sync targets.
3. Any skill previously imported remains on disk in
   `./.agents/skills/` (benign orphan); `.bak/` directories may also
   remain and can be deleted manually.

---

## Dependencies

- `github.com/charmbracelet/bubbletea` — already in `go.mod`
- `github.com/charmbracelet/bubbles` — already in `go.mod`
- `github.com/charmbracelet/x/exp/teatest` — already used in TUI tests
- `gopkg.in/yaml.v3` — already in `go.mod`
- `github.com/spf13/cobra` — already in `go.mod`
- `internal/mesh.NewDefaultResolver` — consumed as-is, no modification
- `internal/writer.AtomicWrite` — consumed as-is, no modification

No new external dependencies are anticipated.

---

## Success Criteria

1. `skill-sync import <skill-name>` copies the full skill directory from
   the winning source into `./.agents/skills/<skillID>/` and exits `0`.
2. Running the same command twice is idempotent (same content → no error,
   no change to the directory).
3. When a skill already exists locally it is overwritten and a `.bak/`
   backup of the previous state is created.
4. When a skill ID exists in multiple sources, the CLI silently picks the
   winner according to the existing resolver rules; the TUI prompts the
   user to choose.
5. When `skill-sync.yaml` is absent, the command exits with a clear error
   message suggesting `skill-sync init`.
6. `skill-sync verify` and `skill-sync sync` are unaffected and continue
   to pass all existing tests after this change lands.
7. `go test ./...` passes with zero failures on Linux; Windows-only
   failure paths are guarded with `t.Skip`.
8. All new packages have table-driven tests; all new TUI views have
   `teatest` golden file coverage.

---

## Open Questions

None. All design decisions were closed with the user prior to this
proposal (see Closed Design Context in the session prompt).

---

## Stakeholder Decisions (summary)

| # | Decision |
|---|---|
| 1 | Source: `targets:` from `skill-sync.yaml` via existing config resolver |
| 2 | Unit: full skill directory, not only `SKILL.md` |
| 3 | Destination v1: `./.agents/skills/<skillID>/` only |
| 4 | Imported skills do NOT participate in mesh sync or `verify` |
| 5 | Conflict in CLI → auto-pick winner; in TUI → interactive choice |
| 6 | TUI: agent-filter screen → skill-list screen → output_view |
| 7 | CLI: `skill-sync import <skill-name>` — one skill, no `--all` |
| 8 | Overwrite with `.bak/` backup of previous skill directory |
| 9 | Multi-destination, `--all`, dry-run, mesh-fix — all deferred |
