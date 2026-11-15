// Package version exposes the toolkit version and build information.
package version

import "runtime/debug"

// Version is the semantic version of go-pilot.
const Version = "1.0.0"

// String returns the module version injected by the Go toolchain when the
// package is built as a dependency, falling back to Version for local builds.
func String() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return Version
}
