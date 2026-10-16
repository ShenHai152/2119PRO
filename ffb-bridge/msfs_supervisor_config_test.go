package main

import (
	"path/filepath"
	"testing"
)

func TestResolveMSFSAdapterPath(t *testing.T) {
	path, enabled, err := resolveMSFSAdapterPath("auto", filepath.Join("C:", "PXN", "PXN2119FFBBridge.exe"))
	if err != nil || !enabled || filepath.Base(path) != "PXNMSFSAdapter.exe" {
		t.Fatalf("auto path=%q enabled=%t err=%v", path, enabled, err)
	}
	if path, enabled, err = resolveMSFSAdapterPath("off", "bridge.exe"); err != nil || enabled || path != "" {
		t.Fatalf("off path=%q enabled=%t err=%v", path, enabled, err)
	}
}
