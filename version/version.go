package version

import "fmt"

const (
	VersionMajor = 0          // Major version component of the current release
	VersionMinor = 2          // Minor version component of the current release
	VersionPatch = 0          // Patch version component of the current release
	VersionMeta  = "unstable" // Version metadata to append to the version string
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	Commit string
	Date   string
)

// Version holds the textual version string.
var Version = func() string {
	return fmt.Sprintf("%d.%d.%d", VersionMajor, VersionMinor, VersionPatch)
}()

// VersionWithMeta holds the textual version string including the metadata.
var VersionWithMeta = func() string {
	v := Version
	if VersionMeta != "" {
		v += "-" + VersionMeta
	}
	return v
}()

func VersionWithCommit() string {
	vsn := VersionWithMeta
	if len(Commit) >= 8 {
		vsn += "-" + Commit[:8]
	}
	if (VersionMeta != "stable") && (Date != "") {
		vsn += "-" + Date
	}
	return vsn
}
