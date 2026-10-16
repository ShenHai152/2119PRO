package winproc

import "strings"

// ClassifyMSFSExecutable returns the process family used only to decide when
// the adapter should be running. The adapter determines the exact 2020/2024
// version from SIMCONNECT_RECV_OPEN after it connects.
func ClassifyMSFSExecutable(name string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "flightsimulator2024.exe":
		return "MSFS 2024", true
	case "flightsimulator.exe":
		return "Microsoft Flight Simulator", true
	default:
		return "", false
	}
}
