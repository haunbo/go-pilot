// Package version exposes the go-pilot toolkit version.
package version

import (
	"runtime/debug"
	"strings"
)

// modulePath is this module's path, used to locate it inside the build info of
// the program that links it.
const modulePath = "github.com/haunbo/go-pilot"

// Version is the semantic version of go-pilot, used when the build info carries
// no release version (for example `go test` inside this repository).
const Version = "1.0.0"

// String returns the go-pilot module version resolved from the build info of the
// program that links this package.
//
// It deliberately does NOT return info.Main.Version: when go-pilot is imported
// as a dependency, the main module is the host service, so Main.Version is the
// service's version, not this library's. For the host program's own version use
// runtime/debug.ReadBuildInfo().Main.Version directly.
func String() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return Version
	}
	// Built from inside this repository: go-pilot is the main module and its
	// version is "(devel)", so no release tag is available.
	if info.Main.Path == modulePath && isRelease(info.Main.Version) {
		return normalize(info.Main.Version)
	}
	for _, dep := range info.Deps {
		if dep.Path != modulePath {
			continue
		}
		if dep.Replace != nil {
			// A replace without a version points at a local checkout, so the
			// real release version is unknown: the require version is a
			// placeholder such as v0.0.0. Fall back to the constant.
			if isRelease(dep.Replace.Version) {
				return normalize(dep.Replace.Version)
			}
			break
		}
		if isRelease(dep.Version) {
			return normalize(dep.Version)
		}
	}
	return Version
}

func isRelease(v string) bool {
	return v != "" && v != "(devel)"
}

func normalize(v string) string {
	return strings.TrimPrefix(v, "v")
}
