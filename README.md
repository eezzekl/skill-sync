# skill-sync

**skill-sync** keeps your AI agent skills in sync across all your tools — Claude Code, Cursor, Gemini CLI, OpenCode, Codex, Windsurf, GitHub Copilot, and more.

It uses a mesh/P2P strategy: no "source of truth" directory. The skill with the highest version (or newest modification time) wins, and it propagates to every other target automatically.

## Install

### Option A — Go install (requires Go 1.21+)

```sh
go install github.com/ezzek/skill-sync/cmd/skill-sync@latest
```

The binary is placed in `$GOPATH/bin` (usually `~/go/bin`). Make sure that directory is in your `PATH`.

```sh
# Add to ~/.bashrc, ~/.zshrc, or equivalent:
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Option B — Download pre-built binary

Go to the [Releases page](https://github.com/ezzek/skill-sync/releases) and download the archive for your platform:

| Platform | File |
|---|---|
| Linux amd64 | `skill-sync_vX.X.X_linux_amd64.tar.gz` |
| Linux arm64 | `skill-sync_vX.X.X_linux_arm64.tar.gz` |
| macOS amd64 (Intel) | `skill-sync_vX.X.X_darwin_amd64.tar.gz` |
| macOS arm64 (Apple Silicon) | `skill-sync_vX.X.X_darwin_arm64.tar.gz` |
| Windows amd64 | `skill-sync_vX.X.X_windows_amd64.zip` |
| Windows arm64 | `skill-sync_vX.X.X_windows_arm64.zip` |

Extract and place the binary somewhere in your `PATH`.

## Quick start

1. Copy the example config and edit it for your setup:

```sh
cp skill-sync.example.yaml skill-sync.yaml
```

2. Edit `skill-sync.yaml` — list every skills directory you want to keep in sync:

```yaml
targets:
  - ~/.claude
  - ~/.cursor
  - ~/.gemini
```

> [!NOTE]
> La tilde (`~`) se expande automáticamente a tu directorio Home en todas las plataformas. Además, el motor es sumamente flexible: puedes especificar tanto el directorio raíz del agente (ej. `~/.claude`) como el subdirectorio de skills directo (ej. `~/.claude/skills`). Ambos formatos son válidos.

3. Run the interactive TUI (no arguments):

```sh
skill-sync
```

Or use a subcommand directly:

```sh
# Sync all skills
skill-sync sync -c skill-sync.yaml

# Check for drift without writing anything
skill-sync verify -c skill-sync.yaml
```

## How it works

- **Scanner** — finds all `SKILL.md` files under each target directory.
- **Resolver** — picks the winner per skill using: version > mtime > conflict.
- **Engine** — writes atomically (temp file + fsync + rename) with a `.bak` backup before overwriting.

Conflicts (same version + same mtime + different content) are skipped and reported — never silently overwritten.

## Supported agents

| Agent | Local dir | Global dir |
|---|---|---|
| Claude Code | `.claude` | `~/.claude` |
| Cursor | `.cursor` | `~/.cursor` |
| Gemini CLI | `.gemini` | `~/.gemini` |
| OpenCode | `.opencode` | `~/.config/opencode` |
| Codex | `.codex` | `~/.codex` |
| Windsurf | `.codeium/windsurf` | `~/.codeium/windsurf` |
| GitHub Copilot | `.copilot` | `~/.copilot` |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for the issue-first workflow, label system, and PR rules.

## License

MIT — see [LICENSE](LICENSE) for details.
