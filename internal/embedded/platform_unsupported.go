//go:build !(linux && amd64)

package embedded

import (
	"embed"
	"runtime"
)

// platformLibs is intentionally empty on unsupported platforms. This keeps
// compilation working even when we do not ship embedded libraries for a
// platform.
var platformLibs embed.FS

const (
	platformSupported = false
	platformLibDir    = ""
)

var (
	platformOS   = runtime.GOOS
	platformArch = runtime.GOARCH
)
