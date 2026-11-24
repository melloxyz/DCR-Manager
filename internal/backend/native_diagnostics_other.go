//go:build !windows

package backend

import "os"

func isReparse(info os.FileInfo) bool { return info.Mode()&os.ModeSymlink != 0 }
