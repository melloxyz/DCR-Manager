//go:build !windows

package backend

import "os"

func creationTicks(os.FileInfo) (uint64, bool) { return 0, false }
