//go:build windows

package backend

import (
	"errors"
	"golang.org/x/sys/windows"
	"unsafe"
)

func nativeParentTimes(rows []nativeCandidate) (map[int]uint64, error) {
	wanted := map[int]bool{}
	for _, r := range rows {
		if dcrEntry(r.Args) && r.Parent > 0 {
			wanted[r.Parent] = true
		}
	}
	result := map[int]uint64{}
	if len(wanted) == 0 {
		return result, nil
	}
	snap, e := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if e != nil {
		return nil, e
	}
	defer windows.CloseHandle(snap)
	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	if e = windows.Process32First(snap, &entry); e != nil {
		return nil, e
	}
	for {
		pid := int(entry.ProcessID)
		if wanted[pid] {
			result[pid] = 0
			h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, entry.ProcessID)
			if err == nil {
				created, ce := birth(h)
				status, se := windows.WaitForSingleObject(h, 0)
				windows.CloseHandle(h)
				if ce == nil && se == nil && status == uint32(windows.WAIT_TIMEOUT) {
					result[pid] = created
				}
			}
		}
		e = windows.Process32Next(snap, &entry)
		if errors.Is(e, windows.ERROR_NO_MORE_FILES) {
			break
		}
		if e != nil {
			return nil, e
		}
	}
	return result, nil
}
