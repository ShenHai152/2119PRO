//go:build windows

package winproc

import (
	"fmt"
	"syscall"
	"unsafe"
)

const (
	th32csSnapProcess = 0x00000002
	maxPath           = 260
)

var (
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	procCreateToolhelpSnapshot = kernel32.NewProc("CreateToolhelp32Snapshot")
	procProcess32FirstW        = kernel32.NewProc("Process32FirstW")
	procProcess32NextW         = kernel32.NewProc("Process32NextW")
	procCloseHandle            = kernel32.NewProc("CloseHandle")
)

type Process struct {
	PID  uint32
	Name string
}

type processEntry32 struct {
	Size            uint32
	Usage           uint32
	ProcessID       uint32
	DefaultHeapID   uintptr
	ModuleID        uint32
	Threads         uint32
	ParentProcessID uint32
	PriClassBase    int32
	Flags           uint32
	ExecutableFile  [maxPath]uint16
}

func snapshotProcesses() ([]Process, error) {
	handle, _, callErr := procCreateToolhelpSnapshot.Call(th32csSnapProcess, 0)
	if handle == ^uintptr(0) {
		return nil, fmt.Errorf("CreateToolhelp32Snapshot: %w", callErr)
	}
	defer procCloseHandle.Call(handle)

	entry := processEntry32{Size: uint32(unsafe.Sizeof(processEntry32{}))}
	ok, _, callErr := procProcess32FirstW.Call(handle, uintptr(unsafe.Pointer(&entry)))
	if ok == 0 {
		return nil, fmt.Errorf("Process32FirstW: %w", callErr)
	}
	processes := make([]Process, 0, 128)
	for {
		processes = append(processes, Process{
			PID:  entry.ProcessID,
			Name: syscall.UTF16ToString(entry.ExecutableFile[:]),
		})
		entry.Size = uint32(unsafe.Sizeof(processEntry32{}))
		ok, _, _ = procProcess32NextW.Call(handle, uintptr(unsafe.Pointer(&entry)))
		if ok == 0 {
			break
		}
	}
	return processes, nil
}

func FindMSFS() (Process, bool, error) {
	processes, err := snapshotProcesses()
	if err != nil {
		return Process{}, false, err
	}
	for _, process := range processes {
		if _, ok := ClassifyMSFSExecutable(process.Name); ok {
			return process, true, nil
		}
	}
	return Process{}, false, nil
}

func Alive(pid uint32) (bool, error) {
	if pid == 0 {
		return true, nil
	}
	processes, err := snapshotProcesses()
	if err != nil {
		return false, err
	}
	for _, process := range processes {
		if process.PID == pid {
			return true, nil
		}
	}
	return false, nil
}
