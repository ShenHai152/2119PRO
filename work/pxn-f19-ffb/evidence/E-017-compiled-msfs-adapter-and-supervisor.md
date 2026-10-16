# E-017 — Compiled MSFS adapter and automatic lifecycle supervisor

- title: Compiled MSFS 2020/2024 adapter and automatic lifecycle supervisor
- observed_at: 2026-08-03T19:10:00+08:00
- source_type: command
- source_ref: `ffb-bridge/cmd/pxn-msfs-adapter`, `msfs_supervisor_windows.go`, `internal/winproc`
- content_hash: `PXN2119FFBBridge.exe=62a8c54780e7d53da170e5d1652496414c703366017191ea47c711eef4ccd6b4`; `PXNMSFSAdapter.exe=08c60d2255e68360b6f01c046e8b64282301df0e9e7e2892590e1814046aa9fe`; `PXN2119-FFB-Bridge.zip=4eafc27ba6fde1a3e30b6fc374de8c55eea472e68f96fa7b9322c9fcaa8e377f`
- repro_command: |
    `go run -buildvcs=false ./cmd/pxn-msfs-adapter --self-test`
    `go test -count=1 -cover -buildvcs=false ./...`
    `go vet ./...`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXNMSFSAdapter.exe ./cmd/pxn-msfs-adapter`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXN2119FFBBridge.exe .`
    `sha256sum PXN2119FFBBridge.exe PXNMSFSAdapter.exe ../PXN2119-FFB-Bridge.zip`
- raw_excerpt: |
    Self-test PASS: MSFS 2020/2024 detection
    Self-test PASS: Pause -> STOP(stick=0, throttle=0)
    Self-test PASS: SimStop -> STOP(stick=0, throttle=0)
    Self-test PASS: Quit -> STOP(stick=0, throttle=0) + exit

    The release contains a Windows amd64 PXNMSFSAdapter.exe that dynamically
    loads SimConnect.dll and therefore does not require SDK headers or a static
    SimConnect.lib during compilation. The bridge monitors
    FlightSimulator.exe and FlightSimulator2024.exe, starts the adapter beside
    the bridge, and prevents repeated child instances. The adapter identifies
    2020/2024 from SIMCONNECT_RECV_OPEN, sends 500 ms heartbeats, sends STOP on
    Pause/SimStop/Quit, exits on Quit, and uses process-liveness fallbacks for
    simulator crashes and parent bridge termination.
    PXNF/1 acknowledgement datagrams are drained non-blockingly so long flights
    cannot fill the adapter's UDP receive buffer.
- linked_workitem: WI-013
- supersedes: none

Real SimConnect dispatch and simulator shutdown behavior require acceptance on
a Windows installation of MSFS 2020 and MSFS 2024. `SimConnect.dll` is a
Microsoft runtime dependency and is not redistributed in this package.
