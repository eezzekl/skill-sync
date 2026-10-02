# Tasks: `skill-import`

**Change ID:** skill-import  
**Branch:** feat/import_skills  
**Date:** 2026-06-05  
**Spec:** `openspec/changes/skill-import/spec.md`  
**Design:** `openspec/changes/skill-import/design.md`

---

## Phase 1 — Foundation

### T-01 · `internal/models/skill.go` — add `Description` field

**Files:** `internal/models/skill.go`, `internal/models/skill_test.go` (if exists; else add to registry_test or create)

**Steps:**
1. Add `Description string \`yaml:"description"\`` to `SkillMetadata`.
2. Add/extend table-driven test to verify `description` round-trips through `yaml.Unmarshal` / `yaml.Marshal`.

**Done when:** `go test ./internal/models/...` passes.

---

### T-02 · `internal/importer/errors.go` — sentinel errors

**Files:** `internal/importer/errors.go` (new)

**Steps:**
1. Create file with `package importer`.
2. Declare `var ErrSkillNotFound = errors.New("skill not found in any configured source")`.
3. Declare `var ErrConflict = errors.New("conflict: identical version and mtime in multiple sources")`.

**Done when:** file compiles (`go build ./internal/importer/...`).

---

### T-03 · `internal/importer/manifest.go` — `DirHash`

**Files:** `internal/importer/manifest.go` (new), `internal/importer/manifest_test.go` (new)

**Steps:**

RED:
1. Write `manifest_test.go` with table-driven cases:
   - `DirHash` is deterministic: same dir hashed twice → same result.
   - `DirHash` is order-independent: two dirs with same files written in different order → same hash.
   - `DirHash` changes when a file's content changes.
   - `DirHash` changes when a file is added.
   - Symlinks are skipped (Unix only; skip on Windows via `runtime.GOOS`).
2. Run tests → confirm RED.

GREEN:
3. Implement `DirHash` in `manifest.go` (design §3):
   - `filepath.WalkDir` → collect regular files.
   - `filepath.ToSlash(rel)` for cross-platform normalisation.
   - Sort paths lexicographically.
   - SHA-256 over `relPath\n` + file bytes for each entry.
   - Return hex string.
4. Run tests → confirm GREEN.

**Done when:** `go test ./internal/importer/...` (manifest tests only) passes.

---

### T-04 · `internal/importer/importer.go` — core importer

**Files:** `internal/importer/importer.go` (new), `internal/importer/importer_test.go` (new)

**Steps:**

RED:
1. Write `importer_test.go` with all table-driven cases from spec §8.1:

   Unit — `FindSkill`:
   - Returns candidates from multiple targets.
   - Returns empty slice for unknown skillID.
   - Skips targets whose `skills/` dir does not exist.
   - Reads YAML frontmatter (`name`, `description`, `version`).

   Unit — `Resolve`:
   - Picks highest version winner.
   - Picks newest mtime when versions tie.
   - Returns `conflict=true` + `ErrConflict` on true tie.

   Unit — `FindAllSkills`:
   - Returns one entry per skillID (not one per candidate).
   - Marks `IsConflict=true` when sources tie.
   - Result is sorted by `SkillID`.
   - Skips targets whose `skills/` dir does not exist.

   Integration — `CopySkill`:
   - Copies full directory tree into destRoot.
   - Creates `.bak` when destination already exists.
   - Removes stale `.bak` before creating new one.
   - Is idempotent on identical source (overwrite = same content).
   - On write failure: returns error and leaves existing destination untouched (inject a read-only file in `.tmp` path to trigger failure).

2. Run tests → confirm RED.

GREEN:
3. Implement `importer.go` (design §4):
   - `SkillCandidate` struct with `IsConflict bool`.
   - `Importer` struct + `New`.
   - `readMetadata` (frontmatter parser aligned with design §4 — `HasPrefix("---\n")` / `"---\r\n"`, `endIndex+3` index math).
   - `FindSkill`.
   - `FindAllSkills` (group by skillID → resolve → mark conflict or use winner).
   - `Resolve` (delegate to `mesh.NewDefaultResolver`, re-map winner back to `SkillCandidate` by `AgentDir`).
   - `CopySkill` (transactional: `.tmp` → commit → rotate `.bak`; design §4 / FR-6).
4. Run tests → GREEN.

**Done when:** `go test ./internal/importer/...` passes with zero failures.

---

## Phase 2 — TUI Views

### T-05 · `internal/tui/import_source_view` — agent-filter screen

**Files:** `internal/tui/import_source_view/model.go` (new), `internal/tui/import_source_view/model_test.go` (new), `testdata/golden/*.txt` (new)

**Steps:**

RED:
1. Write `model_test.go` with `teatest` golden-file tests:
   - `initial.txt` — all sources checked, cursor on first row.
   - `one_source_unchecked.txt` — first source toggled off.
   - `all_unchecked.txt` — all sources deselected (`A` key).
2. Run tests → RED (golden files missing).

GREEN:
3. Implement `model.go` (design §6):
   - `ConfirmedMsg{Sources []string}` and `BackMsg{}`.
   - `Model` wrapping `shared_toggle`.
   - `New(targets []string) Model` — all items checked.
   - `Update`: `enter`/`ctrl+s` → `ConfirmedMsg`; `q`/`esc` → `BackMsg`; `a`/`A` → select/deselect all; delegate rest to `shared_toggle`.
   - `View`: title + toggle + help line.
4. Run tests with `-update` flag to capture golden files.
5. Run tests without `-update` → GREEN.

**Done when:** `go test ./internal/tui/import_source_view/...` passes.

---

### T-06 · `internal/tui/import_skill_view` — skill-select screen

**Files:** `internal/tui/import_skill_view/model.go` (new), `internal/tui/import_skill_view/model_test.go` (new), `testdata/golden/*.txt` (new)

**Steps:**

RED:
1. Write `model_test.go` with `teatest` golden-file tests:
   - `initial.txt` — list rendered with cursor on first row.
   - `skill_selected.txt` — cursor moved to second row.
   - `no_skills.txt` — handled by `output_view` (note in test: `import_skill_view.New` is never called with empty slice; test can verify `len(candidates) > 0` invariant).
   - One test for a candidate with `IsConflict=true` renders `[CONFLICT]` prefix.
2. Run tests → RED.

GREEN:
3. Implement `model.go` (design §7):
   - `ConfirmedMsg{Candidate importer.SkillCandidate}` and `BackMsg{}`.
   - `Model` with `candidates []importer.SkillCandidate`, `cursor int`, `scroll int`.
   - `New(candidates []importer.SkillCandidate) Model`.
   - `Update`: up/down navigation with scroll; `enter` → `ConfirmedMsg`; `esc` → `BackMsg`; `ctrl+c` → quit.
   - `View`: title + count + scrollable list (design §7 column format + `→` cursor marker + `[CONFLICT]` prefix when `IsConflict`).
4. Capture golden files → run clean → GREEN.

**Done when:** `go test ./internal/tui/import_skill_view/...` passes.

---

### T-07 · `internal/tui/menu/model.go` — add Import item

**Files:** `internal/tui/menu/model.go`, `internal/tui/menu/model_test.go`

**Steps:**
1. Add `item{title: "Import", desc: "Import a skill from a configured agent source"}` between `"Verify"` and `"Config"` in the `items` slice.
2. Extend existing menu test (or add case) to verify `"Import"` appears in the list and emits `MenuSelectionMsg{Selection: "Import"}` on Enter.

**Done when:** `go test ./internal/tui/menu/...` passes.

---

### T-08 · `internal/tui/program.go` — wire import states

**Files:** `internal/tui/program.go`, `internal/tui/program_test.go`

**Steps:**
1. Add imports for `importer`, `import_source_view`, `import_skill_view`.
2. Add state constants `stateImportSource`, `stateImportSkill`.
3. Extend `Callbacks` struct with `FindSkillsForImport` and `RunImport` (design §8.3).
4. Add `importSourceModel` and `importSkillModel` fields to `RootModel`.
5. Add `loadImportTargets()` helper (design §8.9).
6. In `Update`:
   - Handle `menu.MenuSelectionMsg{Selection: "Import"}` → call `loadImportTargets`, init `import_source_view`.
   - Handle `import_source_view.ConfirmedMsg` → call `FindSkillsForImport`, transition to `stateImportSkill` or `stateOutput` (empty).
   - Handle `import_source_view.BackMsg` → back to `stateMenu`.
   - Handle `import_skill_view.ConfirmedMsg` → call `RunImport`, transition to `stateOutput`.
   - Handle `import_skill_view.BackMsg` → back to `stateImportSource`.
   - Add dispatch cases for `stateImportSource` and `stateImportSkill`.
7. In `View`: add cases for `stateImportSource` and `stateImportSkill`.
8. Extend `program_test.go`: verify that selecting `"Import"` menu item transitions to `stateImportSource` (use a stub `FindSkillsForImport` callback that returns a fixed list).

**Done when:** `go test ./internal/tui/...` passes.

---

## Phase 3 — CLI & Wiring

### T-09 · `internal/cli/import.go` — Cobra command

**Files:** `internal/cli/import.go` (new), `internal/cli/import_test.go` (new)

**Steps:**

RED:
1. Write `import_test.go` with integration test cases (spec §8.2):
   - Happy path: creates temp dir with `skill-sync.yaml` + source skill dir; runs command; asserts exit 0 + output `"Imported <id> from <path>"` + destination files exist.
   - Missing config: no `skill-sync.yaml` → exit 1, message contains `"skill-sync init"`.
   - Skill not found: config exists but skill not in targets → exit 1, message contains `"not found"`.
   - Wrong arg count (0 args, 2 args) → exit 1 with usage error.
   - Conflict: two sources with identical version+mtime+different hash → exit 1, message contains `"conflict"`.
2. Run tests → RED.

GREEN:
3. Implement `import.go` (design §10): `NewImportCmd` + `runImport` (resolve config → expand targets → `FindSkill` → `Resolve` → `CopySkill` → print confirmation).
4. Run tests → GREEN.

**Done when:** `go test ./internal/cli/... -run TestImport` passes.

---

### T-10 · `internal/cli/root.go` — register command + wire callbacks

**Files:** `internal/cli/root.go`

**Steps:**
1. Add `cmd.AddCommand(NewImportCmd())` in `NewRootCmd`.
2. Add `"os"`, `"path/filepath"`, `"fmt"`, `"github.com/eezzekl/skill-sync/internal/importer"` imports.
3. Wire `FindSkillsForImport` and `RunImport` callbacks in the default `RunE` (design §11.2):
   - `FindSkillsForImport`: build `importer.New(sources, destRoot)` from `sources` arg + `cwd`; return `FindAllSkills()`.
   - `RunImport`: guard `candidate.IsConflict → ErrConflict`; build `importer.New(nil, destRoot)`; call `CopySkill`.

**Done when:** `go test ./internal/cli/...` passes (all CLI tests, not just import).

---

## Phase 4 — Full Suite Verification

### T-11 · Full test suite + regression check

**Steps:**
1. `go build ./...` — clean build, zero errors.
2. `go test ./...` — zero failures on Linux.
3. `go vet ./...` — clean.
4. Confirm `go test ./internal/sync/... ./internal/mesh/... ./internal/cli/... -run TestSync -run TestVerify` still green (spec FR non-regression).
5. Manual smoke test (optional): `go run ./cmd/skill-sync import <some-skill> -c <path>` in a scratch dir.

**Done when:** all tests green, build clean.

---

## Checklist

- [x] T-01 `models.SkillMetadata.Description` + test
- [x] T-02 `importer/errors.go`
- [x] T-03 `importer/manifest.go` + tests (DirHash)
- [x] T-04 `importer/importer.go` + tests (FindSkill, FindAllSkills, Resolve, CopySkill)
- [x] T-05 `tui/import_source_view` + golden tests
- [x] T-06 `tui/import_skill_view` + golden tests
- [x] T-07 `tui/menu` — Import item
- [x] T-08 `tui/program.go` — wire states
- [x] T-09 `cli/import.go` + integration tests
- [x] T-10 `cli/root.go` — register + callbacks
- [x] T-11 Full suite green
