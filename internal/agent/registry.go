package agent

import (
	"gopkg.in/yaml.v3"
	"io"
)

// Config represents the parsed contents of skill-sync.yaml.
type Config struct {
	IncludeGlobs []string `yaml:"include"`
	ExcludeGlobs []string `yaml:"exclude"`
	Targets      []string `yaml:"targets"`
}

// ParseConfig reads YAML configuration from an io.Reader and returns a Config pointer.
func ParseConfig(r io.Reader) (*Config, error) {
	var cfg Config
	decoder := yaml.NewDecoder(r)
	if err := decoder.Decode(&cfg); err != nil && err != io.EOF {
		return nil, err
	}
	return &cfg, nil
}

// ResolvePath replaces a leading tilde (~) with the provided home directory path.
// Handles both Unix (~/) and Windows (~\) separators.
func ResolvePath(path, home string) string {
	if len(path) >= 2 && (path[:2] == "~/" || path[:2] == `~\`) {
		return home + path[1:]
	}
	if path == "~" {
		return home
	}
	return path
}

// ExpandTargets expands tildes (~) in config target directories using the provided home path.
func (c *Config) ExpandTargets(home string) {
	if home == "" {
		return
	}
	for i, t := range c.Targets {
		c.Targets[i] = ResolvePath(t, home)
	}
}
