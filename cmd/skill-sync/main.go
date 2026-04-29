package main

import (
	"os"

	"github.com/ezzek/skill-sync/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
