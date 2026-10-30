// Package version holds build metadata for tinysa_go.
//
// Values are overridden at build time with:
//
//	go build -ldflags "-X tinysa_go/internal/version.Version=v0.1.0 \
//	  -X tinysa_go/internal/version.Commit=$(git rev-parse --short HEAD) \
//	  -X tinysa_go/internal/version.Date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
package version

var (
	// Version is the semantic version of the application.
	Version = "dev"
	// Commit is the git commit hash the binary was built from.
	Commit = "none"
	// Date is the UTC build timestamp.
	Date = "unknown"
)

// String returns a human-readable version string.
func String() string {
	return Version + " (" + Commit + ", " + Date + ")"
}
