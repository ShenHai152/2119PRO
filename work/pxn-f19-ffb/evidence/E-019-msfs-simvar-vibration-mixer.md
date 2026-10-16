# E-019 — MSFS SimVar-driven two-channel vibration mixer

- title: MSFS SimVar telemetry now produces stick and throttle motor amplitudes
- observed_at: 2026-08-03T20:05:00+08:00
- source_type: command
- source_ref: `ffb-bridge/cmd/pxn-msfs-adapter/telemetry.go`, `main_windows.go`, `msfs_supervisor_windows.go`
- content_hash: `PXN2119FFBBridge.exe=aad5e718e84b3c057f0afa30c103dbf4e64ad670596542e1913baa2961666aeb`; `PXNMSFSAdapter.exe=2872a89527d52e4497cb41476abb738597ebe20fc6921f6ef93f05fff2b91a84`; `PXN2119-FFB-Bridge.zip=4a705de46009cdc14e77affc2b461513053167402b761afeeacd7ff6ff2b9637`
- repro_command: |
    `go test -count=1 -cover -buildvcs=false ./...`
    `go vet ./...`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -buildvcs=false -o /tmp/pxn-msfs-telemetry.test.exe ./cmd/pxn-msfs-adapter`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXNMSFSAdapter.exe ./cmd/pxn-msfs-adapter`
- raw_excerpt: |
    The prior adapter only forwarded PXNF/1 ClientData. MSFS does not create
    those amplitudes by itself, so a session could show OPEN/Pause/SimStop while
    both motors remained zero. The adapter now subscribes to twelve SimVars at
    SIM_FRAME frequency and emits final PXNF/1 SET commands every 50 ms.

    Stick channel: G load, acceleration changes, stall warning, ground roll,
    and a small engine-transfer component.
    Throttle channel: GENERAL ENG RPM, TURB ENG N1, combustion, and minor
    ground/acceleration coupling.

    Unit tests validate engine-to-throttle, ground-roll-to-stick, strong stall,
    clamping, smoothing reset, lifecycle STOP, and version identity. External
    aircraft/WASM ClientData SET/STOP commands temporarily override telemetry.
- linked_workitem: WI-014
- supersedes: none

Real MSFS 2020 output amplitudes and subjective tuning require the user to run
the rebuilt adapter in flight. Pause, SimStop, and Quit remain immediate STOP
paths and reset the telemetry mixer.
