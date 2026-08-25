package version

// Single Source of Truth for Pogonian Framework Versioning.
const (
	// Current is the active semantic version of the Pogonian framework and CLI.
	Current = "0.2.0"

	// SpecVersion is the active schema version for .pogonian/architecture/app.ai
	SpecVersion = "0.2.0"

	// MinGoVersion is the minimum supported Go toolchain version.
	MinGoVersion = "1.22"

	// ReleaseName is the codename for this release cycle.
	ReleaseName = "Radix & Shield"
)

// Info returns formatted version string with build details.
func Info() string {
	return "Pogonian Framework v" + Current + " (" + ReleaseName + ")"
}
