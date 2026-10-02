# openspec/ — SDD Artifacts for skill-sync

This directory holds the source of truth for the **Spec-Driven
Development (SDD)** workflow used in `skill-sync`.

## Layout

```text
openspec/
├── config.yaml      # SDD/OpenSpec configuration (test runner, store, review budget)
├── project.md       # Architectural context every phase agent reads first
├── AGENTS.md        # Operating instructions for SDD phase agents
├── README.md        # This file
├── specs/           # Canonical source specs (long-lived, per-capability)
└── changes/         # Active proposals/specs/designs/tasks (per-change folder)
```

## Lifecycle of a Change

```text
init ─► explore ─► proposal ─► spec ─► design ─► tasks ─► apply ─► verify ─► archive
```

Each change lives under `openspec/changes/<change-name>/` and contains:

- `proposal.md`
- `spec.md`
- `design.md`
- `tasks.md`
- (optional) `verify-report.md`, `apply-progress.md`

When a change is archived, its delta is merged into `openspec/specs/`
and the change folder is moved to `openspec/changes/archived/`.

## Conventions

- Strict TDD: every implementation phase records RED/GREEN/TRIANGULATE
  /REFACTOR evidence.
- Atomic writes only — phase agents must use `writer.AtomicWrite`
  when modifying Go source files via the apply phase.
- Review budget: 400 changed lines per PR by default. Tasks phase
  forecasts the diff and asks before exceeding the budget.
- Single PR per change by default; chained PRs only when explicitly
  approved.

See `openspec/AGENTS.md` for phase-by-phase operating instructions.
