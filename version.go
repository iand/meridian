package meridian

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var rawVersion string

var Version string

func init() {
	Version = rawVersion
	if idx := strings.Index(Version, "\n"); idx > -1 {
		Version = Version[:idx]
	}
}
