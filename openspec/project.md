# skill-sync — Project Context for SDD

This document is the source of architectural truth that every SDD phase
agent (`sdd-proposal`, `sdd-spec`, `sdd-design`, `sdd-tasks`,
`sdd-apply`, `sdd-verify`, `sdd-archive`) must read before producing
artifacts. It complements — does not replace — `AGENTS.md` at the repo
root.

## 1. Purpose

`skill-sync` is a **multidirectional (mesh / P2P) CLI + TUI** that keeps
AI agent skills consistent across multiple tool directories on a single
machine (e.g. `~/.claude/skills`, `~/.cursor/skills`,
`~/.pi/agent/skills`, `~/.opencode/skills`, `~/.gemini/skills`). It is
not a publisher and not a marketplace.

## 2. Architecture

Core pipeline:

```text
Scanner  ─►  Resolver  ─►  Engine
 finds        decides       writes
SKILL.md     "winner"      atomically
```

Modules (`internal/`):

- `agent/`            — config parsing, registry, discovery, tilde
                        expansion (`ResolvePath`, `ExpandTargets`).
- `agent/discovery/`  — detect installed agent directories.
- `agent/config_resolver/` — locate `skill-sync.yaml` (cwd, user dir,
                        global fallback).
- `mesh/scanner.go`   — walks each target's `skills/` directory looking
                        for `SKILL.md` files. **Known limitation:**
                        currently only the file is tracked, not the
                        full skill directory (see
                        `context/pending-fixes.md`).
- `mesh/resolver.go`  — applies winner rules:
    1. hash match across all targets → no-op,
    2. highest YAML-frontmatter `version` wins,
    3. on tie, newest `mtime` wins,
    4. equal version + equal mtime + different hash → conflict,
       skip skill and report.
- `sync/engine.go`    — orchestrates Scanner → Resolver → Writer.
- `writer/atomic.go`  — strictly atomic writes: temp file + `fsync` +
                        `os.Rename`, plus `.bak` backup before
                        overwriting any existing target file.
- `cli/`              — Cobra commands: `init`, `config`, `sync`,
                        `verify`. `root.go` sets `SilenceUsage: true`
                        so drift errors do not flood the terminal with
                        usage text.
- `tui/`              — Bubble Tea program with vistas: `menu`,
                        `init_view`, `config_view`, `sync_select_view`,
                        `shared_toggle`, `output_view`. Test harness
                        uses `teatest` + golden files.

Mandatory invariants:

- All file writes go through `writer.AtomicWrite` (including config
  writes from CLI and TUI).
- `verify` (`internal/cli/verify.go`) is strictly read-only and exits
  with code `1` on detected drift.
- Tilde (`~`) in config targets is expanded via `cfg.ExpandTargets(home)`
  at every CLI/TUI entry point.

## 3. Configuration

Schema (`skill-sync.yaml`):

```yaml
include: []          # optional include globs (relative skill names)
exclude: []          # optional exclude globs
targets:             # FLAT LIST OF DIRECTORY PATHS (strings, not objects)
  - ~/.claude
  - ~/.cursor
  - ~/.opencode
  - ~/.pi/agent
  - ~/.gemini
```

Parser: `internal/agent/registry.go` (`ParseConfig`,
`ResolvePath`, `ExpandTargets`).

Target semantics: each `target` is the **parent** of the agent's
`skills/` directory. The scanner concatenates `skills/` internally.

## 4. Testing Discipline

- Runner: `go test ./...` (coverage: `go test -cover ./...`).
- **Strict TDD is mandatory** for non-trivial work: RED → GREEN →
  TRIANGULATE → REFACTOR. Phase agents must record evidence.
- Table-driven tests are mandatory. Pattern:
  `tests := []struct{ name string; ... }{...}`.
- Integration tests for `Engine` and CLI commands use real temp
  directories (`t.TempDir()`); avoid mocking the filesystem.
- TUI tests use Bubble Tea's `teatest` + golden files.
- OS-specific failure paths (e.g. renaming over a read-only dir) may
  be guarded with `t.Skip` + `runtime.GOOS == "windows"`. Do not
  force brittle OS-level injections.

## 5. CLI Surface (current)

- `skill-sync init   -c <path>` — populate config interactively.
- `skill-sync config -c <path>` — adjust config.
- `skill-sync sync   -c <path>` — apply mesh sync across targets.
- `skill-sync verify -c <path>` — read-only drift check; exit 1 on
  drift. `SilenceUsage: true` is set on every command.

## 6. TUI Surface (current)

Entrypoint: `internal/tui/program.go`. Views are independently testable
with `teatest` + golden snapshots stored beside each view's
`model_test.go`.

## 7. Pending Architectural Fixes

Tracked outside of `openspec/changes/` until promoted:

- `context/pending-fixes.md` — mesh sync currently propagates only
  `SKILL.md`, not the full skill directory. Must be addressed in a
  follow-up SDD change that reuses the directory-copy primitive
  introduced by the `import` change.

## 8. Skill Registry

`.atl/skill-registry.md` indexes user/global skills. The parent
orchestrator selects matching `SKILL.md` paths and injects them into
subagent prompts under `## Skills to load before work`. Highly
relevant for this project:

- `go-testing` — Bubbletea teatest, golden files, Go test patterns.
- `lsp-navigation`, `ast-grep` — code intelligence and AST-aware edits.
- `git-expert`, `git-committer`, `work-unit-commits` — commits and PR
  slicing.
- `cognitive-doc-design` — review-facing docs.

## 9. Conventions for Generated Artifacts

- Code, identifiers, comments, commit messages, filenames, and CLI
  help text default to English.
- User-facing conversation in this repo may be in Spanish (Rioplatense
  voseo) but generated artifacts stay in English unless explicitly
  requested otherwise.
- Atomic writes only; never `os.Create` or `os.WriteFile` directly on
  a destination path that already exists.

## 10. Safety

- Never commit unless the user explicitly asks.
- Never run destructive git operations without confirmation.
- Single-writer discipline: keep writes single-threaded unless
  isolated worktrees are explicitly approved.
