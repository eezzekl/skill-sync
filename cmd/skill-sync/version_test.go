package main

import "testing"

// TestResolveVersion covers both ways this binary gets built. GoReleaser stamps
// the version through -ldflags, but `go install` cannot receive a maintainer's
// ldflags: it compiles from the module proxy with its own flags. In that path the
// only version available is the one the Go toolchain embeds in the binary, which
// is why the ldflag value alone is not enough.
func TestResolveVersion(t *testing.T) {
	tests := []struct {
		name          string
		ldflagVersion string
		moduleVersion string
		want          string
	}{
		{
			name:          "ldflag wins when GoReleaser stamped it",
			ldflagVersion: "1.1.0",
			moduleVersion: "v1.1.0+dirty",
			want:          "1.1.0",
		},
		{
			name:          "falls back to module version on go install",
			ldflagVersion: devVersion,
			moduleVersion: "v1.1.0",
			want:          "v1.1.0",
		},
		{
			// The toolchain stamps VCS state by default, so a local build in a
			// git worktree yields a pseudo-version rather than "(devel)". It is
			// reported as-is on purpose: it pins the exact commit and flags a
			// dirty tree, which is precisely what a bug report needs.
			name:          "reports the pseudo-version of a local VCS-stamped build",
			ldflagVersion: devVersion,
			moduleVersion: "v1.1.1-0.20261002231447-f9c2b733d696+dirty",
			want:          "v1.1.1-0.20261002231447-f9c2b733d696+dirty",
		},
		{
			// Reachable with -buildvcs=false.
			name:          "stays dev for a build with no VCS stamping",
			ldflagVersion: devVersion,
			moduleVersion: "(devel)",
			want:          devVersion,
		},
		{
			name:          "stays dev when build info is unavailable",
			ldflagVersion: devVersion,
			moduleVersion: "",
			want:          devVersion,
		},
		{
			name:          "falls back when the ldflag is stamped empty",
			ldflagVersion: "",
			moduleVersion: "v1.1.0",
			want:          "v1.1.0",
		},
		{
			name:          "reports dev when neither source knows the version",
			ldflagVersion: "",
			moduleVersion: "",
			want:          devVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveVersion(tt.ldflagVersion, tt.moduleVersion)
			if got != tt.want {
				t.Errorf("resolveVersion(%q, %q) = %q, want %q",
					tt.ldflagVersion, tt.moduleVersion, got, tt.want)
			}
		})
	}
}
