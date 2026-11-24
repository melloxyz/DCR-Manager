//go:build windows

package backend

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unsafe"

	"dcr-manager/internal/app"
	"golang.org/x/sys/windows"
)

func birth(h windows.Handle) (uint64, error) {
	var c, e, k, u windows.Filetime
	if err := windows.GetProcessTimes(h, &c, &e, &k, &u); err != nil {
		return 0, err
	}
	return uint64(c.HighDateTime)<<32 | uint64(c.LowDateTime), nil
}

func commandArgs(h windows.Handle) ([]string, error) {
	// ProcessCommandLineInformation (60) returns a UNICODE_STRING and its text.
	buf := make([]byte, 128*1024)
	var n uint32
	if err := windows.NtQueryInformationProcess(h, 60, unsafe.Pointer(&buf[0]), uint32(len(buf)), &n); err != nil {
		return nil, err
	}
	u := (*windows.NTUnicodeString)(unsafe.Pointer(&buf[0]))
	if u.Length == 0 {
		return nil, nil
	}
	base := uintptr(unsafe.Pointer(&buf[0]))
	ptr := uintptr(unsafe.Pointer(u.Buffer))
	if u.Length%2 != 0 || ptr < base || ptr-base > uintptr(len(buf)) || uintptr(u.Length) > uintptr(len(buf))-(ptr-base) {
		return nil, errors.New("invalid process command line")
	}
	text := u.String()
	runtime.KeepAlive(buf)
	return windows.DecomposeCommandLine(text)
}

func nativeRows() ([]nativeCandidate, error) {
	snap, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(snap)
	var e windows.ProcessEntry32
	e.Size = uint32(unsafe.Sizeof(e))
	if err = windows.Process32First(snap, &e); err != nil {
		return nil, err
	}
	var out []nativeCandidate
	for {
		if strings.EqualFold(windows.UTF16ToString(e.ExeFile[:]), "node.exe") {
			h, openErr := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.SYNCHRONIZE, false, e.ProcessID)
			if openErr != nil {
				if !errors.Is(openErr, windows.ERROR_INVALID_PARAMETER) {
					return nil, fmt.Errorf("cannot inspect Node process %d: %w", e.ProcessID, openErr)
				}
			} else {
				created, ce := birth(h)
				args, ae := commandArgs(h)
				state, waitErr := windows.WaitForSingleObject(h, 0)
				windows.CloseHandle(h)
				if waitErr != nil {
					return nil, errors.New("cannot verify Node process state")
				}
				if state != windows.WAIT_OBJECT_0 {
					if ce != nil || ae != nil || created == 0 || len(args) == 0 {
						return nil, fmt.Errorf("cannot verify Node process %d", e.ProcessID)
					}
					out = append(out, nativeCandidate{ID: app.ProcessIdentity{PID: int(e.ProcessID), Created: created}, Parent: int(e.ParentProcessID), Args: args})
				}
			}
		}
		err = windows.Process32Next(snap, &e)
		if errors.Is(err, windows.ERROR_NO_MORE_FILES) {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}

func nativeRemote() ([]nativeCandidate, error) {
	rows, err := nativeRows()
	if err != nil {
		return nil, err
	}
	return classifyRemote(rows), nil
}

// The retained OS handle binds termination to the same process object even
// if its PID is recycled. Preflight all targets before terminating any.
type boundProcess struct {
	candidate nativeCandidate
	handle    windows.Handle
}

func bindTarget(r nativeCandidate) (windows.Handle, error) {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION|windows.PROCESS_TERMINATE|windows.SYNCHRONIZE, false, uint32(r.ID.PID))
	if err != nil {
		return 0, err
	}
	c, ce := birth(h)
	a, ae := commandArgs(h)
	if ce != nil || ae != nil || c != r.ID.Created || !dcrEntry(a) || !equalArgs(a, r.Args) {
		windows.CloseHandle(h)
		return 0, errors.New("process identity changed")
	}
	return h, nil
}
func equalArgs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func terminateBound(ctx context.Context, h windows.Handle) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	state, err := windows.WaitForSingleObject(h, 0)
	if err != nil {
		return err
	}
	if state == windows.WAIT_OBJECT_0 {
		return nil
	}
	if err = windows.TerminateProcess(h, 1); err != nil {
		state, _ = windows.WaitForSingleObject(h, 0)
		if state == windows.WAIT_OBJECT_0 {
			return nil
		}
		return err
	}
	state, err = windows.WaitForSingleObject(h, 5000)
	if err != nil {
		return err
	}
	if state != windows.WAIT_OBJECT_0 {
		return errors.New("process did not exit")
	}
	return nil
}
func stopExact(ctx context.Context, targets []nativeCandidate, verify func() error) error {
	handles := make([]boundProcess, 0, len(targets))
	defer func() {
		for _, x := range handles {
			windows.CloseHandle(x.handle)
		}
	}()
	for _, r := range targets {
		h, err := bindTarget(r)
		if err != nil {
			return fmt.Errorf("preflight refused PID %d: %w", r.ID.PID, err)
		}
		handles = append(handles, boundProcess{r, h})
	}
	if err := verify(); err != nil {
		return err
	}
	// Parent processes precede their later-created workers.
	sort.SliceStable(handles, func(i, j int) bool { return handles[i].candidate.ID.Created < handles[j].candidate.ID.Created })
	for _, x := range handles {
		if err := terminateBound(ctx, x.handle); err != nil {
			return fmt.Errorf("stop incomplete; PID %d: %w", x.candidate.ID.PID, err)
		}
	}
	return nil
}

type launchSpec struct{ node, entry, dir string }

func prepareLaunch() (launchSpec, error) {
	var s launchSpec
	root := os.Getenv("LOCALAPPDATA")
	if root == "" || !filepath.IsAbs(root) {
		return s, errors.New("LOCALAPPDATA unavailable")
	}
	s.dir = filepath.Join(root, "DesktopCommander")
	s.entry = filepath.Join(s.dir, "node_modules", "@wonderwhy-er", "desktop-commander", "dist", "index.js")
	i, err := os.Stat(s.entry)
	if err != nil || !i.Mode().IsRegular() {
		return s, errors.New("dedicated DCR installation missing")
	}
	s.node, err = exec.LookPath("node.exe")
	if err != nil {
		return s, errors.New("Node.js unavailable")
	}
	s.node, err = filepath.Abs(s.node)
	return s, err
}
func launchRemote(s launchSpec) (windows.Handle, app.ProcessIdentity, error) {
	var id app.ProcessIdentity
	exe, err := windows.UTF16PtrFromString(s.node)
	if err != nil {
		return 0, id, err
	}
	cmd, err := windows.UTF16PtrFromString(windows.ComposeCommandLine([]string{s.node, s.entry, "remote"}))
	if err != nil {
		return 0, id, err
	}
	dir, err := windows.UTF16PtrFromString(s.dir)
	if err != nil {
		return 0, id, err
	}
	var si windows.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi windows.ProcessInformation
	if err = windows.CreateProcess(exe, cmd, nil, nil, false, windows.CREATE_NEW_CONSOLE, nil, dir, &si, &pi); err != nil {
		return 0, id, err
	}
	windows.CloseHandle(pi.Thread)
	created, err := birth(pi.Process)
	if err != nil {
		windows.CloseHandle(pi.Process)
		return 0, id, err
	}
	return pi.Process, app.ProcessIdentity{PID: int(pi.ProcessId), Created: created}, nil
}

func operateNative(parent context.Context, a app.RemoteAction, expected []app.ProcessIdentity) error {
	ctx, cancel := context.WithTimeout(parent, 35*time.Second)
	defer cancel()
	// Mutex ownership is thread-affine on Windows.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	name, _ := windows.UTF16PtrFromString("Local\\DCRManager")
	mutex, err := windows.CreateMutex(nil, false, name)
	if err != nil && !errors.Is(err, windows.ERROR_ALREADY_EXISTS) {
		return err
	}
	if mutex == 0 {
		return errors.New("maintenance lock unavailable")
	}
	defer windows.CloseHandle(mutex)
	state, err := windows.WaitForSingleObject(mutex, 0)
	if err != nil {
		return err
	}
	if state != windows.WAIT_OBJECT_0 && state != windows.WAIT_ABANDONED {
		return errors.New("another maintenance manager is open")
	}
	defer windows.ReleaseMutex(mutex)
	list, err := nativeRemote()
	if err != nil {
		return err
	}
	if !sameIdentities(expected, identities(list)) {
		return errors.New("Remote process identities changed; refresh first")
	}
	var spec launchSpec
	if a != app.Stop {
		spec, err = prepareLaunch()
		if err != nil {
			return err
		}
	}
	if a != app.Start {
		err = stopExact(ctx, list, func() error {
			again, e := nativeRemote()
			if e != nil {
				return e
			}
			if !sameIdentities(expected, identities(again)) {
				return errors.New("Remote process set changed before stop")
			}
			return nil
		})
		if err != nil {
			return err
		}
		again, e := nativeRemote()
		if e != nil {
			return e
		}
		if len(again) > 0 {
			return errors.New("Remote still running; no further action")
		}
	}
	if a == app.Stop {
		return nil
	}
	if err = ctx.Err(); err != nil {
		return err
	}
	again, err := nativeRemote()
	if err != nil {
		return err
	}
	if len(again) > 0 {
		return errors.New("Remote started elsewhere; no duplicate launched")
	}
	h, id, err := launchRemote(spec)
	if err != nil {
		return err
	}
	defer windows.CloseHandle(h)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if err = ctx.Err(); err != nil {
			return err
		}
		state, e := windows.WaitForSingleObject(h, 400)
		if e != nil {
			return e
		}
		if state == windows.WAIT_OBJECT_0 {
			return errors.New("Remote exited during launch; inspect its console")
		}
		rows, e := nativeRemote()
		if e != nil {
			return e
		}
		for _, r := range rows {
			if r.ID == id {
				return nil
			}
		}
	}
	return errors.New("Remote launch not verified; refresh before retrying")
}
