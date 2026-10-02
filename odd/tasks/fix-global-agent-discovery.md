# Fix global agent discovery in `init` and `config`

## Problem

`internal/cli/init.go:88` and `internal/cli/config.go:65` inject the user *config*
directory into the discovery *home* seam:

```go
d.UserHomeDir = func() (string, error) { return getUserConfig() }
```

`getUserConfig` is `os.UserConfigDir` (`~/.config` on Linux), so `Discovery.Scan()`
resolves every global agent directory to `~/.config/.pi/agent`, `~/.config/.claude`,
etc. None of those exist, so no global agent is ever discovered.

Pi is fully invisible because its `AgentConfig.LocalDir` is `""`
(`internal/agent/discovery/discovery.go:52`): it has only a global path.

The TUI is unaffected — `internal/tui/program.go:71,86` call `discovery.New()`
without overriding the seam.

## Root cause

One package-level variable (`getUserConfig`) serves two distinct responsibilities:
locating `skill-sync.yaml` and locating the user home for agent discovery.

## Fix

Split the seam: keep `getUserConfig` for config location, add `getUserHome`
(`os.UserHomeDir`) for discovery.

## Tasks

- [x] 1. RED: test that `init` discovers a global-only agent (Pi) when home and
      config directories differ.
- [x] 2. GREEN: add the `getUserHome` seam and wire it in `init.go`.
- [x] 3. RED: equivalent test for `config`.
- [x] 4. GREEN: wire `getUserHome` in `config.go`.
- [x] 5. Make existing `init`/`config` tests hermetic by stubbing `getUserHome`,
      so they cannot read the developer's real home directory.
- [x] 6. Full suite + manual CLI verification against the real home.

## Evidence

**RED (task 1)** — `go test ./internal/cli/ -run TestInitDiscoversGlobalAgents`:
all three cases emitted an empty `targets:` list.

**RED (task 3)** — `go test ./internal/cli/ -run TestConfigDiscoversGlobalAgents`:
both cases emitted `targets: []`.

**Task 5 was not cosmetic.** After wiring the real home seam, the pre-existing
case `TestInitCmd/no_agents_found_prints_'no_skills_configured'` failed because it
started discovering the developer's actual `~/.claude`, `~/.gemini` and
`~/.pi/agent`. Those tests had been passing only because the broken seam pointed
them at a directory that never existed. Both suites now pin `getUserHome` to an
empty temporary directory.

**GREEN** — `go test ./...`: all 14 packages pass. `go vet ./...` clean,
`gofmt -l internal/cli/` empty.

**Manual** — `XDG_CONFIG_HOME=/tmp/pitest/cfg skill-sync init` against the real
home now produces:

```yaml
targets:
  - /home/eezzekl/.config/opencode/skills
  - /home/eezzekl/.claude/skills
  - /home/eezzekl/.gemini/skills
  - /home/eezzekl/.copilot/skills
  - /home/eezzekl/.pi/agent/skills
```

Previously this was an empty `targets:` list.

## Notes

`~/.pi/agent/skills` does not exist yet. `Discovery.Scan` matches on the agent
root (`~/.pi/agent`) and `init` appends `/skills`; `internal/sync/engine.go:43-56`
creates missing target directories, so the first `sync` materializes it.

Pi project-local skills remain out of scope: `AgentConfig.LocalDir` for Pi is
`""` by design.
