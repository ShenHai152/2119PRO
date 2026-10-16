package main

import (
	"fmt"
	"path/filepath"
	"strings"
)

func resolveMSFSAdapterPath(setting, bridgeExecutable string) (string, bool, error) {
	setting = strings.TrimSpace(setting)
	switch strings.ToLower(setting) {
	case "", "auto":
		if bridgeExecutable == "" {
			return "", false, fmt.Errorf("resolve bridge executable before locating PXNMSFSAdapter.exe")
		}
		return filepath.Join(filepath.Dir(bridgeExecutable), "PXNMSFSAdapter.exe"), true, nil
	case "off", "disabled", "none":
		return "", false, nil
	default:
		path, err := filepath.Abs(setting)
		if err != nil {
			return "", false, fmt.Errorf("resolve MSFS adapter path %q: %w", setting, err)
		}
		return path, true, nil
	}
}
