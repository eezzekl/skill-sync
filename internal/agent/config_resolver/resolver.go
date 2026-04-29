package config_resolver

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type Resolver struct {
	Getwd         func() (string, error)
	UserConfigDir func() (string, error)
	Stderr        io.Writer
}

func New() *Resolver {
	return &Resolver{
		Getwd:         os.Getwd,
		UserConfigDir: os.UserConfigDir,
		Stderr:        os.Stderr,
	}
}

func (r *Resolver) Resolve(flagPath string) (string, error) {
	// 1. Flag precedence
	if flagPath != "" {
		if _, err := os.Stat(flagPath); err == nil {
			return flagPath, nil
		}
		return "", fmt.Errorf("config file at flag path %s not found", flagPath)
	}

	// 2. Local precedence (CWD)
	if r.Getwd != nil {
		if cwd, err := r.Getwd(); err == nil {
			localPath := filepath.Join(cwd, "skill-sync.yaml")
			if _, err := os.Stat(localPath); err == nil {
				fmt.Fprintf(r.Stderr, "using local config %s\n", localPath)
				return localPath, nil
			}
		}
	}

	// 3. Global precedence (User Config Dir)
	if r.UserConfigDir != nil {
		if userDir, err := r.UserConfigDir(); err == nil {
			globalPath := filepath.Join(userDir, "skill-sync", "skill-sync.yaml")
			if _, err := os.Stat(globalPath); err == nil {
				return globalPath, nil
			}
		}
	}

	return "", fmt.Errorf("no config file found")
}
