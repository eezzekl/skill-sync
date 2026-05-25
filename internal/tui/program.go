package tui

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/ezzek/skill-sync/internal/agent"
	"github.com/ezzek/skill-sync/internal/agent/config_resolver"
	"github.com/ezzek/skill-sync/internal/agent/discovery"
	"github.com/ezzek/skill-sync/internal/models"
	"github.com/ezzek/skill-sync/internal/tui/config_view"
	"github.com/ezzek/skill-sync/internal/tui/init_view"
	"github.com/ezzek/skill-sync/internal/tui/menu"
	"github.com/ezzek/skill-sync/internal/tui/output_view"
	"github.com/ezzek/skill-sync/internal/tui/sync_select_view"
	"github.com/ezzek/skill-sync/internal/writer"
	"gopkg.in/yaml.v3"
)

type state int

const (
	stateMenu state = iota
	stateInit
	stateConfig
	stateSyncSelect
	stateOutput
)

type Callbacks struct {
	ScanSkills func() ([]models.SkillSyncInfo, error)
	RunSync    func(skillFilter []string) (string, error)
	RunVerify  func() (string, error)
}

type RootModel struct {
	state           state
	menuModel       menu.Model
	initModel       tea.Model
	cfgModel        tea.Model
	syncSelectModel tea.Model
	outModel        tea.Model

	callbacks Callbacks
}

func NewRootModel(callbacks Callbacks) RootModel {
	return RootModel{
		state:     stateMenu,
		menuModel: menu.New(),
		callbacks: callbacks,
	}
}

func (m RootModel) Init() tea.Cmd {
	return m.menuModel.Init()
}

func (m RootModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case menu.MenuSelectionMsg:
		switch msg.Selection {
		case "Init":
			d := discovery.New()
			targets, _ := d.Scan()
			var existing []string
			if cfgPath, err := config_resolver.New().Resolve(""); err == nil {
				if f, err := os.Open(cfgPath); err == nil {
					cfg, _ := agent.ParseConfig(f)
					existing = cfg.Targets
					f.Close()
				}
			}
			m.initModel = init_view.New(targets, existing)
			m.state = stateInit
			return m, m.initModel.Init()

		case "Config":
			d := discovery.New()
			targets, _ := d.Scan()
			var existing []string
			if cfgPath, err := config_resolver.New().Resolve(""); err == nil {
				if f, err := os.Open(cfgPath); err == nil {
					cfg, _ := agent.ParseConfig(f)
					existing = cfg.Targets
					f.Close()
				}
			}
			m.cfgModel = config_view.New(targets, existing)
			m.state = stateConfig
			return m, m.cfgModel.Init()

		case "Sync":
			skillIDs, err := m.callbacks.ScanSkills()
			if err != nil {
				m.outModel = output_view.New("Error scanning skills: " + err.Error())
				m.state = stateOutput
				return m, m.outModel.Init()
			}
			m.syncSelectModel = sync_select_view.New(skillIDs)
			m.state = stateSyncSelect
			return m, m.syncSelectModel.Init()

		case "Verify":
			out, err := m.callbacks.RunVerify()
			if err != nil {
				out += "\nError: " + err.Error()
			}
			m.outModel = output_view.New(out)
			m.state = stateOutput
			return m, m.outModel.Init()
		}

	case init_view.InitConfigMsg:
		out := saveConfig(msg.Targets)
		m.outModel = output_view.New(out)
		m.state = stateOutput
		return m, m.outModel.Init()
	case init_view.BackMsg:
		m.state = stateMenu
		return m, m.menuModel.Init()

	case config_view.UpdateConfigMsg:
		out := saveConfig(msg.Targets)
		m.outModel = output_view.New(out)
		m.state = stateOutput
		return m, m.outModel.Init()
	case config_view.BackMsg:
		m.state = stateMenu
		return m, m.menuModel.Init()

	case sync_select_view.SyncSelectedMsg:
		out, err := m.callbacks.RunSync(msg.SkillIDs)
		if err != nil {
			out += "\nError: " + err.Error()
		}
		m.outModel = output_view.New(out)
		m.state = stateOutput
		return m, m.outModel.Init()
	case sync_select_view.BackMsg:
		m.state = stateMenu
		return m, m.menuModel.Init()

	case output_view.DoneMsg:
		m.state = stateMenu
		return m, m.menuModel.Init()
	}

	var cmd tea.Cmd
	switch m.state {
	case stateMenu:
		var updated tea.Model
		updated, cmd = m.menuModel.Update(msg)
		m.menuModel = updated.(menu.Model)
	case stateInit:
		var updated tea.Model
		updated, cmd = m.initModel.Update(msg)
		if updated != nil {
			m.initModel = updated
		}
	case stateConfig:
		var updated tea.Model
		updated, cmd = m.cfgModel.Update(msg)
		if updated != nil {
			m.cfgModel = updated
		}
	case stateSyncSelect:
		var updated tea.Model
		updated, cmd = m.syncSelectModel.Update(msg)
		if updated != nil {
			m.syncSelectModel = updated
		}
	case stateOutput:
		var updated tea.Model
		updated, cmd = m.outModel.Update(msg)
		if updated != nil {
			m.outModel = updated
		}
	}

	return m, cmd
}

func (m RootModel) View() string {
	switch m.state {
	case stateMenu:
		return m.menuModel.View()
	case stateInit:
		return m.initModel.View()
	case stateConfig:
		return m.cfgModel.View()
	case stateSyncSelect:
		return m.syncSelectModel.View()
	case stateOutput:
		return m.outModel.View()
	}
	return ""
}

func saveConfig(targets []string) string {
	configPath := ""
	if cfgPath, err := config_resolver.New().Resolve(""); err == nil {
		configPath = cfgPath
	} else {
		if userDir, err := os.UserConfigDir(); err == nil {
			configPath = filepath.Join(userDir, "skill-sync", "skill-sync.yaml")
			os.MkdirAll(filepath.Dir(configPath), 0755)
		} else {
			configPath = "skill-sync.yaml"
		}
	}

	root := yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode},
		},
	}

	targetsSeq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, t := range targets {
		skillsPath := t
		if filepath.Base(t) != "skills" {
			skillsPath = filepath.Join(t, "skills")
		}
		targetsSeq.Content = append(targetsSeq.Content, &yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: filepath.ToSlash(skillsPath),
		})
	}
	root.Content[0].Content = append(root.Content[0].Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "targets"},
		targetsSeq,
	)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&root); err != nil {
		return fmt.Sprintf("Failed to save config: %v", err)
	}
	enc.Close()

	if err := writer.AtomicWrite(configPath, buf.Bytes()); err != nil {
		return fmt.Sprintf("Failed to save config atomically to %s: %v", configPath, err)
	}
	return fmt.Sprintf("Configuration saved to %s", configPath)
}

type Program struct {
	callbacks Callbacks
}

func NewProgram(callbacks Callbacks) *Program {
	return &Program{callbacks: callbacks}
}

func (p *Program) Run() error {
	model := NewRootModel(p.callbacks)
	prg := tea.NewProgram(model, tea.WithAltScreen())
	_, err := prg.Run()
	return err
}
