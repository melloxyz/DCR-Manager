//go:build windows

package backend

import (
	"golang.org/x/sys/windows"
	"os"
	"syscall"
)

func isReparse(info os.FileInfo) bool {
	data, ok := info.Sys().(*syscall.Win32FileAttributeData)
	return !ok || data.FileAttributes&windows.FILE_ATTRIBUTE_REPARSE_POINT != 0
}
