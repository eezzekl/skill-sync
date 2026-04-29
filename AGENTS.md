# AI Agent Instructions

**Project:** `skill-sync` — A multidirectional (Mesh/P2P) CLI tool to synchronize AI agent skills across different tool directories (Claude, Cursor, OpenCode, etc.).

## 🏗 Architecture & Entrypoints
*   **CLI Framework:** Cobra. The entrypoint is `cmd/skill-sync/main.go`.
*   **Core Flow:** `Scanner` (finds SKILL.md files) -> `Resolver` (determines winner) -> `Engine` (writes files).
*   **Mesh Sync Logic (The "Winner" rules):**
    1.  If hashes (SHA-256) match across all targets, it's a no-op.
    2.  If hashes differ, the highest `version` in the YAML frontmatter wins.
    3.  If versions match (or are missing), the newest modification time (`mtime`) wins.
    4.  If versions and mtimes are identical but hashes differ, a conflict is detected and the skill is skipped.
*   **File Writing:** Strictly atomic. Uses temporary files + `fsync` + `os.Rename`. Always creates a `.bak` backup before overwriting an existing file.

## ⚙️ Configuration
*   **Format:** YAML (`skill-sync.yaml`).
*   **Structure:** The `targets` field is a flat list of directory paths (strings), NOT a list of objects.
    ```yaml
    targets:
      - ./.claude/skills
      - ./.cursor/skills
    ```
*   **Parsing:** `internal/agent/registry.go` handles parsing and tilde (`~`) expansion to the home directory.

## 🧪 Testing & TDD Workflow
*   **Runner:** Standard `go test ./...`. Coverage: `go test -cover ./...`.
*   **Style:** Table-driven tests are mandatory. See existing `tests := []struct{...}` in the codebase.
*   **Integration Tests:** The `Engine` and `CLI` commands (`sync`, `verify`) use real temporary directories. Avoid mocking the filesystem where possible.
*   **OS Quirks:** Some file manipulation failure paths (like renaming over read-only directories) are hard to test on Windows. These are guarded with `t.Skip` + `runtime.GOOS == "windows"`. Do not force brittle OS-level tests if they require privileged fault injection.
*   **Verification:** The `verify` command (`internal/cli/verify.go`) is strictly read-only. It detects drift and exits with code 1.

## 🛠 Commands & Execution
*   **Run CLI:** `go run ./cmd/skill-sync [command]`
*   **Commands:** `sync`, `verify` (read-only drift detection). Both require a config file via `-c` or `--config`.
*   **Note:** Cobra usage output is silenced (`SilenceUsage: true`) so that a drift detection error doesn't spam the terminal with the help menu.