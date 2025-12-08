//go:build linux && amd64 && embedded_cuda_portable

package embedded

import "embed"

// platformLibs contains the Linux/amd64 CUDA portable shared libraries that are embedded into the binary.
//
//go:embed libs/linux/amd64/cuda-portable/*.so
var platformLibs embed.FS

const (
	platformOS        = "linux"
	platformArch      = "amd64"
	platformVariant   = "cuda-portable"
	platformPortable  = true
	platformLibExt    = ".so"
	platformSupported = true
	platformLibDir    = "libs/linux/amd64/cuda-portable"
)
