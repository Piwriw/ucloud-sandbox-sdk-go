package sandbox

import (
	"strconv"
	"strings"
)

// envd versions at which a capability became available. A sandbox built from an
// older template reports an older version, and the SDK degrades rather than
// calling something that is not there.
var (
	envdVersionMinimum        = envdVersion{0, 1, 0}
	envdVersionRecursiveWatch = envdVersion{0, 1, 4}
	envdVersionMetrics        = envdVersion{0, 1, 5}
	envdVersionDiskMetrics    = envdVersion{0, 2, 4}
	envdVersionStdin          = envdVersion{0, 3, 0}
	envdVersionDefaultUser    = envdVersion{0, 4, 0}
)

// envdVersion is a major.minor.patch triple.
type envdVersion [3]int

// parseEnvdVersion reads a version string, treating anything unparsable as 0.
// An empty or malformed version therefore sorts below every gate, which fails
// closed: the capability is assumed absent.
func parseEnvdVersion(version string) envdVersion {
	var parsed envdVersion
	parts := strings.Split(strings.TrimPrefix(version, "v"), ".")
	for i := 0; i < 3 && i < len(parts); i++ {
		if n, err := strconv.Atoi(parts[i]); err == nil {
			parsed[i] = n
		}
	}
	return parsed
}

// lessThan reports whether v is older than other.
func (v envdVersion) lessThan(other envdVersion) bool {
	for i := range v {
		if v[i] != other[i] {
			return v[i] < other[i]
		}
	}
	return false
}

// supports reports whether this envd is new enough for a capability.
func (v envdVersion) supports(gate envdVersion) bool {
	return !v.lessThan(gate)
}
