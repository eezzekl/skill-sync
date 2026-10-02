package main

import (
	"os"
	"runtime/debug"

	"github.com/eezzekl/skill-sync/internal/cli"
)

// devVersion is the placeholder reported when no build stamped a real version.
const devVersion = "dev"

// version is set at build time via -ldflags "-X main.version=vX.Y.Z"
var version = devVersion

// resolveVersion decides which version string to report. A GoReleaser build
// stamps ldflagVersion and wins outright. A `go install` build cannot receive
// those ldflags, so it falls back to the module version the toolchain embeds.
// A plain local build has neither and reports devVersion.
func resolveVersion(ldflagVersion, moduleVersion string) string {
	if ldflagVersion != "" && ldflagVersion != devVersion {
		return ldflagVersion
	}
	// "(devel)" is what the toolchain reports for a build outside a tagged
	// module version, which carries no more information than devVersion.
	if moduleVersion != "" && moduleVersion != "(devel)" {
		return moduleVersion
	}
	return devVersion
}

// moduleVersion returns the module version the Go toolchain embeds in the
// binary, or an empty string when build information is unavailable.
func moduleVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	return info.Main.Version
}

func main() {
	if err := cli.NewRootCmd(resolveVersion(version, moduleVersion())).Execute(); err != nil {
		os.Exit(1)
	}
}
