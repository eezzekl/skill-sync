# Design: `skill-import`

**Change ID:** skill-import  
**Branch:** feat/import_skills  
**Date:** 2026-06-05  
**Status:** APPROVED

---

## 1. New File Layout

```
internal/
  importer/
    errors.go            — ErrSkillNotFound, ErrConflict
    manifest.go          — DirHash
    manifest_test.go
    importer.go          — SkillCandidate, Importer, New, FindSkill, Resolve, CopySkill
    importer_test.go
  tui/
    import_source_view/
      model.go
      model_test.go
      testdata/golden/
        initial.txt
        one_source_unchecked.txt
        all_unchecked.txt
    import_skill_view/
      model.go
      model_test.go
      testdata/golden/
        initial.txt
        skill_selected.txt
        no_skills.txt
  cli/
    import.go            — NewImportCmd
    import_test.go
```

Modified existing files:

```
internal/models/skill.go          + Description field on SkillMetadata
internal/tui/menu/model.go        + "Import" item
internal/tui/program.go           + 2 states, 2 model fields, 2 callbacks, message handlers
internal/cli/root.go              + NewImportCmd(), 2 new Callbacks fields
```

---

## 2. `internal/importer/errors.go`

```go
package importer

import "errors"

var ErrSkillNotFound = errors.New("skill not found in any configured source")
var ErrConflict      = errors.New("conflict: identical version and mtime in multiple sources")
```

Both are exported so `internal/cli` and `internal/tui` can use `errors.Is`.

`ErrConfigNotFound` is **not** in this package. It is produced by
`internal/cli/import.go` using the existing `config_resolver` error path
(see §7).

---

## 3. `internal/importer/manifest.go`

```go
package importer

import (
    "crypto/sha256"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
)

// DirHash computes a deterministic, cross-platform SHA-256 over all regular
// files in dir. Symlinks are skipped.
func DirHash(dir string) (string, error) {
    type entry struct{ rel, absPath string }
    var entries []entry

    err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.Type()&fs.ModeSymlink != 0 || d.IsDir() {
            return nil
        }
        rel, err := filepath.Rel(dir, path)
        if err != nil {
            return err
        }
        entries = append(entries, entry{
            rel:     filepath.ToSlash(rel), // cross-platform determinism
            absPath: path,
        })
        return nil
    })
    if err != nil {
        return "", fmt.Errorf("DirHash walk %s: %w", dir, err)
    }

    sort.Slice(entries, func(i, j int) bool {
        return entries[i].rel < entries[j].rel
    })

    h := sha256.New()
    for _, e := range entries {
        data, err := os.ReadFile(e.absPath)
        if err != nil {
            return "", fmt.Errorf("DirHash read %s: %w", e.rel, err)
        }
        h.Write([]byte(e.rel + "\n"))
        h.Write(data)
    }
    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

---

## 4. `internal/importer/importer.go`

```go
package importer

import (
    "errors"
    "fmt"
    "io/fs"
    "os"
    "path/filepath"
    "sort"
    "strings"
    "time"

    "github.com/ezzek/skill-sync/internal/mesh"
    "github.com/ezzek/skill-sync/internal/models"
    "github.com/ezzek/skill-sync/internal/writer"
    "gopkg.in/yaml.v3"
)

// SkillCandidate is a skill directory found during source discovery.
type SkillCandidate struct {
    SkillID    string               // directory name, e.g. "git-expert"
    SourceDir  string               // absolute path to the skill directory
    AgentDir   string               // absolute path to the parent target
    Metadata   models.SkillMetadata // parsed from SKILL.md frontmatter
    Mtime      time.Time            // mtime of SKILL.md
    Hash       string               // DirHash of SourceDir
    IsConflict bool                 // true if multiple sources compete in a tie conflict
}

// Importer drives import operations for a fixed set of source targets
// and a fixed destination root.
type Importer struct {
    Targets  []string // tilde-expanded absolute paths
    DestRoot string   // absolute path to ./.agents/skills/
}

// New returns an Importer.
// targets must be tilde-expanded absolute paths.
// destRoot is typically filepath.Join(cwd, ".agents", "skills").
func New(targets []string, destRoot string) *Importer {
    return &Importer{Targets: targets, DestRoot: destRoot}
}

// FindSkill walks each target's skills/ subdirectory and returns all
// candidates whose directory name matches skillID (case-sensitive).
// Returns an empty slice (not an error) when no match is found.
func (imp *Importer) FindSkill(skillID string) ([]SkillCandidate, error) {
    var candidates []SkillCandidate
    for _, target := range imp.Targets {
        skillsDir := filepath.Join(target, "skills")
        skillDir  := filepath.Join(skillsDir, skillID)
        mdPath    := filepath.Join(skillDir, "SKILL.md")

        info, err := os.Stat(mdPath)
        if errors.Is(err, fs.ErrNotExist) {
            continue
        }
        if err != nil {
            return nil, fmt.Errorf("stat %s: %w", mdPath, err)
        }

        meta, err := readMetadata(mdPath)
        if err != nil {
            meta = models.SkillMetadata{} // tolerate parse errors
        }

        hash, err := DirHash(skillDir)
        if err != nil {
            return nil, fmt.Errorf("hash %s: %w", skillDir, err)
        }

        candidates = append(candidates, SkillCandidate{
            SkillID:   skillID,
            SourceDir: skillDir,
            AgentDir:  target,
            Metadata:  meta,
            Mtime:     info.ModTime(),
            Hash:      hash,
        })
    }
    return candidates, nil
}

// FindAllSkills walks each target's skills/ subdirectory, groups candidates
// by SkillID, and resolves the winner for each unique skill ID. If a skill has
// a true tie conflict, it returns a candidate with IsConflict set to true.
func (imp *Importer) FindAllSkills() ([]SkillCandidate, error) {
    candidatesByDir := make(map[string][]SkillCandidate)
    for _, target := range imp.Targets {
        skillsDir := filepath.Join(target, "skills")
        entries, err := os.ReadDir(skillsDir)
        if errors.Is(err, fs.ErrNotExist) {
            continue
        }
        if err != nil {
            return nil, fmt.Errorf("read %s: %w", skillsDir, err)
        }
        for _, e := range entries {
            if !e.IsDir() {
                continue
            }
            skillID  := e.Name()
            skillDir := filepath.Join(skillsDir, skillID)
            mdPath   := filepath.Join(skillDir, "SKILL.md")
            info, err := os.Stat(mdPath)
            if errors.Is(err, fs.ErrNotExist) {
                continue
            }
            if err != nil {
                continue
            }

            meta, _ := readMetadata(mdPath)
            hash, err := DirHash(skillDir)
            if err != nil {
                return nil, fmt.Errorf("hash %s: %w", skillDir, err)
            }
            c := SkillCandidate{
                SkillID:   skillID,
                SourceDir: skillDir,
                AgentDir:  target,
                Metadata:  meta,
                Mtime:     info.ModTime(),
                Hash:      hash,
            }
            candidatesByDir[skillID] = append(candidatesByDir[skillID], c)
        }
    }

    var results []SkillCandidate
    for _, group := range candidatesByDir {
        winner, conflict, err := Resolve(group)
        if conflict {
            c := group[0]
            c.IsConflict = true
            results = append(results, c)
        } else if err == nil && winner != nil {
            results = append(results, *winner)
        }
    }

    // Sort results by SkillID so list is deterministic.
    sort.Slice(results, func(i, j int) bool {
        return results[i].SkillID < results[j].SkillID
    })

    return results, nil
}

// Resolve converts candidates to SkillInstances and delegates winner
// selection to mesh.NewDefaultResolver(). Returns the winning candidate,
// a conflict flag, and any error.
func Resolve(candidates []SkillCandidate) (*SkillCandidate, bool, error) {
    if len(candidates) == 0 {
        return nil, false, ErrSkillNotFound
    }
    if len(candidates) == 1 {
        c := candidates[0]
        return &c, false, nil
    }

    instances := make([]models.SkillInstance, len(candidates))
    for i, c := range candidates {
        instances[i] = models.SkillInstance{
            ID:        c.SkillID,
            Path:      filepath.Join(c.SourceDir, "SKILL.md"),
            TargetDir: c.AgentDir,
            Hash:      c.Hash,
            Mtime:     c.Mtime,
            Metadata:  c.Metadata,
        }
    }

    winner, conflict, err := mesh.NewDefaultResolver().Resolve(instances)
    if err != nil {
        return nil, false, err
    }
    if conflict {
        return nil, true, ErrConflict
    }

    // Re-map winner back to its SkillCandidate by matching AgentDir.
    for i := range candidates {
        if candidates[i].AgentDir == winner.TargetDir {
            c := candidates[i]
            return &c, false, nil
        }
    }
    return nil, false, fmt.Errorf("importer: resolve winner not found in candidates (internal error)")
}

// CopySkill copies the full skill directory from src into
// filepath.Join(destRoot, skillID) using the transactional strategy:
//   1. Write all files to <skillID>.tmp/
//   2. On failure: cleanup .tmp, return error (existing dir untouched).
//   3. On success: remove stale .bak, rename existing → .bak, rename .tmp → final.
func (imp *Importer) CopySkill(skillID, src string) error {
    if err := os.MkdirAll(imp.DestRoot, 0o755); err != nil {
        return fmt.Errorf("create dest root: %w", err)
    }

    destFinal := filepath.Join(imp.DestRoot, skillID)
    destTmp   := filepath.Join(imp.DestRoot, skillID+".tmp")
    destBak   := filepath.Join(imp.DestRoot, skillID+".bak")

    // Ensure tmp is clean before starting.
    _ = os.RemoveAll(destTmp)

    // Copy all files into .tmp/
    copyErr := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return err
        }
        if d.Type()&fs.ModeSymlink != 0 {
            return nil // skip symlinks silently
        }
        rel, err := filepath.Rel(src, path)
        if err != nil {
            return err
        }
        tmpDest := filepath.Join(destTmp, rel)
        if d.IsDir() {
            return os.MkdirAll(tmpDest, 0o755)
        }
        data, err := os.ReadFile(path)
        if err != nil {
            return fmt.Errorf("read %s: %w", rel, err)
        }
        if err := os.MkdirAll(filepath.Dir(tmpDest), 0o755); err != nil {
            return err
        }
        return writer.AtomicWrite(tmpDest, data)
    })
    if copyErr != nil {
        _ = os.RemoveAll(destTmp) // cleanup on failure
        return fmt.Errorf("copy skill %s: %w", skillID, copyErr)
    }

    // Commit: rotate backup, rename tmp → final.
    _ = os.RemoveAll(destBak)
    if _, err := os.Stat(destFinal); err == nil {
        if err := os.Rename(destFinal, destBak); err != nil {
            _ = os.RemoveAll(destTmp)
            return fmt.Errorf("backup existing skill: %w", err)
        }
    }
    if err := os.Rename(destTmp, destFinal); err != nil {
        // Attempt to restore backup.
        if _, statErr := os.Stat(destBak); statErr == nil {
            _ = os.Rename(destBak, destFinal)
        }
        return fmt.Errorf("commit skill %s: %w", skillID, err)
    }
    return nil
}

// readMetadata parses the YAML frontmatter from a SKILL.md file.
// Returns zero-value SkillMetadata on any parse error.
func readMetadata(mdPath string) (models.SkillMetadata, error) {
    data, err := os.ReadFile(mdPath)
    if err != nil {
        return models.SkillMetadata{}, err
    }
    s := string(data)
    // Check for opening fence with Unix or Windows newline.
    if !strings.HasPrefix(s, "---\n") && !strings.HasPrefix(s, "---\r\n") {
        return models.SkillMetadata{}, nil
    }
    // Find the end of the frontmatter.
    endIndex := strings.Index(s[3:], "\n---")
    if endIndex == -1 {
        return models.SkillMetadata{}, nil
    }
    yamlStr := s[3 : endIndex+3]
    var meta models.SkillMetadata
    _ = yaml.Unmarshal([]byte(yamlStr), &meta)
    return meta, nil
}
```

> **Note:** `FindAllSkills` is added to support the TUI flow (Screen B needs
> unique resolved skills across the filtered sources, or conflicts marked).
> `FindSkill` (single skillID) is kept for the CLI path.

---

## 5. `internal/models/skill.go` — change

Add `Description` to `SkillMetadata`:

```go
type SkillMetadata struct {
    Version     float64 `yaml:"version"`
    Name        string  `yaml:"name"`
    Description string  `yaml:"description"` // NEW
}
```

No other changes to `models`.

---

## 6. `internal/tui/import_source_view/model.go`

```go
package import_source_view

import (
    "path/filepath"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/ezzek/skill-sync/internal/tui/shared_toggle"
    "github.com/ezzek/skill-sync/internal/tui/styles"
)

// ConfirmedMsg carries the subset of source directories the user kept checked.
type ConfirmedMsg struct{ Sources []string }

// BackMsg signals the root model to return to the main menu.
type BackMsg struct{}

// Model wraps shared_toggle for agent-source multi-selection.
type Model struct {
    toggle  shared_toggle.Model
    sources []string // parallel to toggle items; index → full target path
}

// New builds the model from a list of resolved target directories.
// All sources start checked.
func New(targets []string) Model {
    items := make([]shared_toggle.Item, len(targets))
    for i, t := range targets {
        items[i] = shared_toggle.Item{
            Label:    filepath.Base(t) + "  (" + t + ")",
            Value:    t,
            Selected: true,
        }
    }
    return Model{
        toggle:  shared_toggle.New(items),
        sources: targets,
    }
}

func (m Model) Init() tea.Cmd { return m.toggle.Init() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "enter", "ctrl+s":
            var checked []string
            for _, item := range m.toggle.GetSelected() {
                checked = append(checked, item.Value)
            }
            return m, func() tea.Msg { return ConfirmedMsg{Sources: checked} }
        case "q", "esc":
            return m, func() tea.Msg { return BackMsg{} }
        case "ctrl+c":
            return m, tea.Quit
        case "a":
            m.toggle = m.toggle.SelectAll()
            return m, nil
        case "A":
            m.toggle = m.toggle.DeselectAll()
            return m, nil
        }
    }
    var cmd tea.Cmd
    var updated tea.Model
    updated, cmd = m.toggle.Update(msg)
    m.toggle = updated.(shared_toggle.Model)
    return m, cmd
}

func (m Model) View() string {
    var s strings.Builder
    s.WriteString(styles.TitleStyle.Render("Import — Select Sources"))
    s.WriteString("\n\nChoose which agent directories to search:\n")
    s.WriteString(m.toggle.View())
    s.WriteString(styles.HelpStyle.Render(
        "\n[space/enter] toggle • [a] all • [A] none • [enter/ctrl+s] next • [q/esc] back",
    ))
    return s.String()
}
```

---

## 7. `internal/tui/import_skill_view/model.go`

Single-selection cursor-based list. Does **not** use `shared_toggle` (no
checkboxes needed; radio-select semantics).

```go
package import_skill_view

import (
    "fmt"
    "strings"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/ezzek/skill-sync/internal/importer"
    "github.com/ezzek/skill-sync/internal/tui/styles"
)

// ConfirmedMsg carries the skill the user selected for import.
type ConfirmedMsg struct{ Candidate importer.SkillCandidate }

// BackMsg signals the root model to return to import_source_view.
type BackMsg struct{}

const maxVisible = 15
const descMaxLen = 60

type Model struct {
    candidates []importer.SkillCandidate
    cursor     int
    scroll     int
}

// New builds the model from a list of candidates.
// Callers must ensure len(candidates) > 0; use output_view for the empty case.
func New(candidates []importer.SkillCandidate) Model {
    return Model{candidates: candidates}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "up", "k":
            if m.cursor > 0 {
                m.cursor--
                if m.cursor < m.scroll {
                    m.scroll = m.cursor
                }
            }
        case "down", "j":
            if m.cursor < len(m.candidates)-1 {
                m.cursor++
                if m.cursor >= m.scroll+maxVisible {
                    m.scroll = m.cursor - maxVisible + 1
                }
            }
        case "enter":
            c := m.candidates[m.cursor]
            return m, func() tea.Msg { return ConfirmedMsg{Candidate: c} }
        case "esc":
            return m, func() tea.Msg { return BackMsg{} }
        case "ctrl+c":
            return m, tea.Quit
        }
    }
    return m, nil
}

func (m Model) View() string {
    var s strings.Builder
    s.WriteString(styles.TitleStyle.Render("Import — Select Skill"))
    s.WriteString(fmt.Sprintf("\n\n%d skill(s) found — choose one to import:\n", len(m.candidates)))

    n   := len(m.candidates)
    end := m.scroll + maxVisible
    if end > n {
        end = n
    }

    if m.scroll > 0 {
        s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↑ %d more above", m.scroll)) + "\n")
    }
    for i := m.scroll; i < end; i++ {
        c      := m.candidates[i]
        cursor := "  "
        if m.cursor == i {
            cursor = "→ "
        }
        name := c.Metadata.Name
        if name == "" {
            name = c.SkillID
        }
        desc := c.Metadata.Description
        if c.IsConflict {
            desc = "[CONFLICT] " + desc
        }
        if len([]rune(desc)) > descMaxLen {
            desc = string([]rune(desc)[:descMaxLen]) + "…"
        }
        line := fmt.Sprintf("%s%-24s  %-28s  %s", cursor, c.SkillID, name, desc)
        if m.cursor == i {
            s.WriteString(styles.ListItemSelectedStyle.Render(line) + "\n")
        } else {
            s.WriteString(styles.ListItemStyle.Render(line) + "\n")
        }
    }
    if end < n {
        s.WriteString(styles.SubtleStyle.Render(fmt.Sprintf("  ↓ %d more below", n-end)) + "\n")
    }

    s.WriteString(styles.HelpStyle.Render("\n[↑/↓] navigate • [enter] import • [esc] back"))
    return s.String()
}
```

---

## 8. `internal/tui/program.go` — changes

### 8.1 New imports

```go
"github.com/ezzek/skill-sync/internal/importer"
"github.com/ezzek/skill-sync/internal/tui/import_source_view"
"github.com/ezzek/skill-sync/internal/tui/import_skill_view"
```

### 8.2 State constants (add after `stateOutput`)

```go
stateImportSource
stateImportSkill
```

### 8.3 Callbacks struct (add two fields)

```go
type Callbacks struct {
    ScanSkills          func() ([]models.SkillSyncInfo, error)
    RunSync             func(skillFilter []string) (string, error)
    RunVerify           func() (string, error)
    FindSkillsForImport func(sources []string) ([]importer.SkillCandidate, error) // NEW
    RunImport           func(candidate importer.SkillCandidate) (string, error)   // NEW
}
```

### 8.4 RootModel struct (add two model fields)

```go
type RootModel struct {
    // ... existing fields ...
    importSourceModel tea.Model // NEW
    importSkillModel  tea.Model // NEW
    callbacks         Callbacks
}
```

### 8.5 `Update` — new menu case

```go
case "Import":
    targets, err := loadImportTargets()
    if err != nil {
        m.outModel = output_view.New("Error loading config: " + err.Error())
        m.state = stateOutput
        return m, m.outModel.Init()
    }
    m.importSourceModel = import_source_view.New(targets)
    m.state = stateImportSource
    return m, m.importSourceModel.Init()
```

### 8.6 `Update` — new message handlers

```go
case import_source_view.ConfirmedMsg:
    candidates, err := m.callbacks.FindSkillsForImport(msg.Sources)
    if err != nil {
        m.outModel = output_view.New("Error scanning skills: " + err.Error())
        m.state = stateOutput
        return m, m.outModel.Init()
    }
    if len(candidates) == 0 {
        m.outModel = output_view.New("No skills found in the selected sources.")
        m.state = stateOutput
        return m, m.outModel.Init()
    }
    m.importSkillModel = import_skill_view.New(candidates)
    m.state = stateImportSkill
    return m, m.importSkillModel.Init()

case import_source_view.BackMsg:
    m.state = stateMenu
    return m, m.menuModel.Init()

case import_skill_view.ConfirmedMsg:
    out, err := m.callbacks.RunImport(msg.Candidate)
    if err != nil {
        out = "Import error: " + err.Error()
    }
    m.outModel = output_view.New(out)
    m.state = stateOutput
    return m, m.outModel.Init()

case import_skill_view.BackMsg:
    m.state = stateImportSource
    return m, m.importSourceModel.Init()
```

### 8.7 `Update` — dispatch existing state machine (add two cases)

```go
case stateImportSource:
    var updated tea.Model
    updated, cmd = m.importSourceModel.Update(msg)
    if updated != nil {
        m.importSourceModel = updated
    }
case stateImportSkill:
    var updated tea.Model
    updated, cmd = m.importSkillModel.Update(msg)
    if updated != nil {
        m.importSkillModel = updated
    }
```

### 8.8 `View` — add two cases

```go
case stateImportSource:
    return m.importSourceModel.View()
case stateImportSkill:
    return m.importSkillModel.View()
```

### 8.9 New helper function `loadImportTargets`

```go
// loadImportTargets resolves skill-sync.yaml and returns the expanded target list.
func loadImportTargets() ([]string, error) {
    cfgPath, err := config_resolver.New().Resolve("")
    if err != nil {
        return nil, fmt.Errorf("no skill-sync.yaml found; run 'skill-sync init' to create one")
    }
    f, err := os.Open(cfgPath)
    if err != nil {
        return nil, err
    }
    defer f.Close()
    cfg, err := agent.ParseConfig(f)
    if err != nil {
        return nil, err
    }
    if home, err := os.UserHomeDir(); err == nil {
        cfg.ExpandTargets(home)
    }
    return cfg.Targets, nil
}
```

---

## 9. `internal/tui/menu/model.go` — change

Add `"Import"` item between `"Verify"` and `"Config"`:

```go
items := []list.Item{
    item{title: "Init",   desc: "Initialize configuration and discover skills"},
    item{title: "Sync",   desc: "Synchronize skills across directories"},
    item{title: "Verify", desc: "Check for drift without syncing"},
    item{title: "Import", desc: "Import a skill from a configured agent source"}, // NEW
    item{title: "Config", desc: "Configure sync targets"},
    item{title: "Quit",   desc: "Exit the application"},
}
```

---

## 10. `internal/cli/import.go`

```go
package cli

import (
    "errors"
    "fmt"
    "os"
    "path/filepath"

    "github.com/ezzek/skill-sync/internal/agent"
    "github.com/ezzek/skill-sync/internal/agent/config_resolver"
    "github.com/ezzek/skill-sync/internal/importer"
    "github.com/spf13/cobra"
)

func NewImportCmd() *cobra.Command {
    var configPath string

    cmd := &cobra.Command{
        Use:          "import <skill-name>",
        Short:        "Import a skill from a configured agent source into ./.agents/skills/",
        Args:         cobra.ExactArgs(1),
        SilenceUsage: true,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runImport(cmd, configPath, args[0])
        },
    }
    cmd.Flags().StringVarP(&configPath, "config", "c", "", "Path to skill-sync.yaml")
    return cmd
}

func runImport(cmd *cobra.Command, configPath, skillID string) error {
    // 1. Resolve config.
    resolvedPath, err := config_resolver.New().Resolve(configPath)
    if err != nil {
        return fmt.Errorf("no skill-sync.yaml found; run 'skill-sync init' to create one")
    }
    f, err := os.Open(resolvedPath)
    if err != nil {
        return fmt.Errorf("open config: %w", err)
    }
    defer f.Close()

    cfg, err := agent.ParseConfig(f)
    if err != nil {
        return fmt.Errorf("parse config: %w", err)
    }
    if home, err := os.UserHomeDir(); err == nil {
        cfg.ExpandTargets(home)
    }

    // 2. Build importer.
    cwd, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("getwd: %w", err)
    }
    destRoot := filepath.Join(cwd, ".agents", "skills")
    imp := importer.New(cfg.Targets, destRoot)

    // 3. Find candidates.
    candidates, err := imp.FindSkill(skillID)
    if err != nil {
        return err
    }
    if len(candidates) == 0 {
        return fmt.Errorf("skill %q not found in any configured source", skillID)
    }

    // 4. Resolve winner.
    winner, _, err := importer.Resolve(candidates)
    if err != nil {
        if errors.Is(err, importer.ErrConflict) {
            return fmt.Errorf("conflict: %q has identical version and mtime in multiple sources; resolve manually", skillID)
        }
        return err
    }

    // 5. Copy.
    if err := imp.CopySkill(skillID, winner.SourceDir); err != nil {
        return err
    }

    fmt.Fprintf(cmd.OutOrStdout(), "Imported %s from %s\n", skillID, winner.SourceDir)
    return nil
}
```

---

## 11. `internal/cli/root.go` — changes

### 11.1 Add `NewImportCmd()` to the root command

```go
cmd.AddCommand(NewImportCmd())
```

### 11.2 Wire import callbacks in the default `RunE`

```go
callbacks := tui.Callbacks{
    // ... existing fields ...

    FindSkillsForImport: func(sources []string) ([]importer.SkillCandidate, error) {
        cwd, err := os.Getwd()
        if err != nil {
            return nil, err
        }
        destRoot := filepath.Join(cwd, ".agents", "skills")
        imp := importer.New(sources, destRoot)
        return imp.FindAllSkills()
    },

    RunImport: func(candidate importer.SkillCandidate) (string, error) {
        if candidate.IsConflict {
            return "", importer.ErrConflict
        }
        cwd, err := os.Getwd()
        if err != nil {
            return "", err
        }
        destRoot := filepath.Join(cwd, ".agents", "skills")
        imp := importer.New(nil, destRoot) // sources not needed for CopySkill
        if err := imp.CopySkill(candidate.SkillID, candidate.SourceDir); err != nil {
            return "", err
        }
        return fmt.Sprintf("Imported %s from %s", candidate.SkillID, candidate.SourceDir), nil
    },
}
```

New imports needed in `root.go`:

```go
"os"
"path/filepath"
"fmt"
"github.com/ezzek/skill-sync/internal/importer"
```

---

## 12. Dependency Graph

```
internal/importer
    ↑
    ├── internal/mesh (Resolve)
    ├── internal/models (SkillMetadata, SkillInstance)
    ├── internal/writer (AtomicWrite)
    └── gopkg.in/yaml.v3

internal/tui/import_source_view
    ↑
    └── internal/tui/shared_toggle

internal/tui/import_skill_view
    ↑
    └── internal/importer

internal/tui/program.go
    ↑
    ├── internal/importer
    ├── internal/tui/import_source_view
    └── internal/tui/import_skill_view

internal/cli/import.go
    ↑
    ├── internal/importer
    ├── internal/agent
    └── internal/agent/config_resolver

internal/cli/root.go
    ↑
    └── internal/importer (for callback signatures)
```

No circular dependencies. `internal/importer` does not import any `tui` or
`cli` package.

---

## 13. Implementation Order

1. `internal/models/skill.go` — add `Description` field + test.
2. `internal/importer/errors.go` — sentinel errors.
3. `internal/importer/manifest.go` + `manifest_test.go`.
4. `internal/importer/importer.go` + `importer_test.go` (TDD: RED → GREEN).
5. `internal/tui/import_source_view/model.go` + golden tests.
6. `internal/tui/import_skill_view/model.go` + golden tests.
7. `internal/tui/menu/model.go` — add Import item.
8. `internal/tui/program.go` — wire new states.
9. `internal/cli/import.go` + `import_test.go`.
10. `internal/cli/root.go` — add `NewImportCmd()` + import callbacks.
