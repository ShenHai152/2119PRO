# E-016 — Simulator identity and runtime status implementation

- title: Simulator identity and runtime status implementation
- observed_at: 2026-08-03T18:05:00+08:00
- source_type: command
- source_ref: `ffb-bridge/service_core.go`, `service_windows.go`, `main_windows.go`, `adapters/msfs/PXNMSFSAdapter.cpp`, `adapters/xplane/PXNXPlaneAdapter.cpp`
- content_hash: `PXN2119FFBBridge.exe=ff5dbac6974c77bacfb55a8a9430a199d988e48bd75cfb79c622f45e231d725d`; `PXN2119-FFB-Bridge.zip=8aa52d37f32bd1b6fc2f5d6503c00f72cf7161bed6aeb558828fc431e9426a17`
- repro_command: |
    `go test -count=1 -cover -buildvcs=false ./...`
    `go vet ./...`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -buildvcs=false -o /tmp/pxn-status-final.test.exe .`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXN2119FFBBridge.exe .`
    `sha256sum PXN2119FFBBridge.exe ../PXN2119-FFB-Bridge.zip`
- raw_excerpt: |
    PXNF/1 keeps its 16-byte layout and uses the previously reserved flags word
    for adapter identity and paused/stopped/exiting state. The service displays
    device, adapter, simulator state, calibrated stick amplitude, and calibrated
    throttle amplitude. It performs a one-second HID health check, reports
    disconnect/reconnect, marks watchdog stops, and stops after adapter timeout.
    MSFS source subscribes to Pause/SimStart/SimStop and identifies 2020/2024
    from SIMCONNECT_RECV_OPEN. X-Plane source monitors sim/time/paused. Both send
    500 ms idle heartbeats and explicit STOP on exit. Go tests and Windows
    cross-build passed with 61.7% statement coverage.
- linked_workitem: WI-012
- supersedes: none

The simulator adapters require their official SDKs and were not compiled or
runtime-tested in this environment. Their MSFS/X-Plane behavior remains a
hardware/simulator acceptance item.
