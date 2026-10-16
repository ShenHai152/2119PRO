//go:build windows

package main

import (
	"fmt"
	"syscall"
	"unsafe"
)

const moveFileReplaceExisting = 0x1

func replaceFile(source, destination string) error {
	sourcePtr, err := syscall.UTF16PtrFromString(source)
	if err != nil {
		return err
	}
	destinationPtr, err := syscall.UTF16PtrFromString(destination)
	if err != nil {
		return err
	}
	proc := syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")
	ok, _, callErr := proc.Call(
		uintptr(unsafe.Pointer(sourcePtr)),
		uintptr(unsafe.Pointer(destinationPtr)),
		moveFileReplaceExisting,
	)
	if ok == 0 {
		return fmt.Errorf("MoveFileExW: %w", callErr)
	}
	return nil
}
