# openspec/AGENTS.md — Phase Agent Instructions

Operating contract for SDD phase agents working on `skill-sync`.
Read `openspec/project.md` and the repo-root `AGENTS.md` before
producing any artifact.

## Common Contract

Every phase agent must return an envelope containing:

```text
status
executive_summary
artifacts          # paths produced or modified
next_recommended
risks
skill_resolution   # paths-injected | fallback-registry | fallback-path | none
```

Hard rules:

- Atomic writes only via `writer.AtomicWrite` for Go source under
  `internal/` and `cmd/`; for `openspec/` artifacts a normal write is
  acceptable but **never partial overwrites** mid-edit.
- Strict TDD evidence is mandatory in `apply` and `verify` phases.
- Do not delete or rewrite `openspec/specs/` from inside a change
  folder; the `archive` phase performs the merge.
- Generated artifacts default to English regardless of conversation
  language.
- Never commit, push, or open PRs without explicit user approval.

## Phase Cheatsheet

### `sdd-init`
Initialize project SDD context. Writes `openspec/config.yaml`,
`openspec/project.md`, `openspec/AGENTS.md`, `openspec/README.md`.
Detects test runner and records skill registry status. **Done.**

### `sdd-explore`
Optional pre-proposal exploration. Maps relevant files, lists open
design questions. Saves notes under
`openspec/changes/<change>/explore.md`.

### `sdd-proposal`
Writes `openspec/changes/<change>/proposal.md`:
- Goal, Why, Non-goals, Risks, Open Questions, Stakeholder Decisions.
- No code, no spec text — only intent.

### `sdd-spec`
Writes `openspec/changes/<change>/spec.md` as a **delta spec**:
- Requirements with `REQ-<n>` IDs.
- Scenarios in Given/When/Then.
- Explicit references to existing capabilities being modified.

### `sdd-design`
Writes `openspec/changes/<change>/design.md`:
- Concrete technical approach.
- Module/package boundaries, types, signatures.
- Test strategy (table-driven cases, golden files, temp dirs).
- Migration notes if any.

### `sdd-tasks`
Writes `openspec/changes/<change>/tasks.md`:
- Numbered, ordered tasks with acceptance criteria.
- Each task notes whether it produces tests, implementation, or both.
- **Forecasts changed lines**; if total > 400, pauses and surfaces a
  chained-PR question to the parent orchestrator.

### `sdd-apply`
Implements tasks in order with strict TDD evidence:

```text
RED         → failing test committed conceptually first
GREEN       → minimum code to pass
TRIANGULATE → second test exposing a generalization
REFACTOR    → code cleanup, tests still green
```

Records progress under
`openspec/changes/<change>/apply-progress.md`.

### `sdd-verify`
Read-only verification:
- `go test ./...` passes.
- Spec requirements covered by tests.
- No drift between spec and implementation.
Writes `openspec/changes/<change>/verify-report.md`. Exits with
non-zero status on failure.

### `sdd-archive`
Merges verified delta into `openspec/specs/`, moves change folder to
`openspec/changes/archived/<change>/`. Updates any cross-references.

### `sdd-sync` (optional)
Syncs verified delta specs into canonical specs without archiving
the change (useful when multiple changes are in flight).

## Skill Loading

Parent orchestrator passes exact `SKILL.md` paths in the prompt under
`## Skills to load before work`. Phase agents must read those before
producing artifacts. If paths are missing, fall back to
`.atl/skill-registry.md` and report `skill_resolution: fallback-registry`.

## Review Workload Guard

`sdd-tasks` is responsible for forecasting changed lines. If the
forecast exceeds 400, do not proceed to `sdd-apply` silently — surface
the chained-PR decision to the parent and wait for approval.

## Testing Discipline

Project-specific (see `openspec/project.md` §4):

- Table-driven tests are mandatory.
- Real `t.TempDir()` for integration tests; no FS mocks unless
  unavoidable.
- TUI: `teatest` + golden files.
- Strict TDD: every commit-worthy unit has paired test evidence.
