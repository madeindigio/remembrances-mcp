//go:build linux && amd64 && embedded && embedded_cuda

package embedded

import "embed"

// platformLibs contains the Linux/amd64 CUDA shared libraries that are embedded into the binary.
//
//go:embed libs/linux/amd64/cuda/*.so
var platformLibs embed.FS

const (
	platformOS        = "linux"
	platformArch      = "amd64"
	platformVariant   = "cuda"
	platformPortable  = false
	platformLibExt    = ".so"
	platformSupported = true
	platformLibDir    = "libs/linux/amd64/cuda"
)
