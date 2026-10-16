# E-020 — Windows UDP polling and SimStart resume correction

- title: Adapter packets were not visible to the service and telemetry stayed paused
- observed_at: 2026-08-03T20:30:00+08:00
- source_type: log
- source_ref: user bridge/adapter console plus `service_windows.go` and adapter lifecycle source
- content_hash: `PXN2119FFBBridge.exe=1e65ecb6a92d371eacfb9686cc6c84349b4610777c0c0ea34c6e1d61e8f8f246`; `PXNMSFSAdapter.exe=9028829020a0e7563b55df5c38c402e2be6c23ff81afaf324253b7b58b1424d7`; `PXN2119-FFB-Bridge.zip=19543eccd63a6bd09f6ea9630ef6935032f3435936af806d6f2ea98af639f90f`
- repro_command: |
    `PXN2119FFBBridge.exe --mode service --msfs-telemetry-gain 2`
    `go test -count=1 -cover -buildvcs=false ./...`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -buildvcs=false -o /tmp/pxn-msfs-udpfix.test.exe .`
- raw_excerpt: |
    The real run showed SimConnect OPEN and lifecycle events in the child
    adapter while the bridge remained adapter=none/simulator=waiting. The
    service used SetReadDeadline(time.Now()) immediately before ReadFromUDP;
    on Windows an already-expired deadline can return WSAETIMEDOUT before a
    queued datagram is checked. The deadline now extends one millisecond into
    the future.

    The same run delivered Pause -> SimStop -> SimStart without a subsequent
    resume event. SimStart now clears the adapter's paused state, allowing
    telemetry output to resume. Adapter diagnostics now report service ACK and
    first telemetry frame, or explicit timeouts for either path.
- linked_workitem: WI-014
- supersedes: E-019

The code paths and Windows builds are validated offline. Real Windows
acceptance must confirm the new `PXNF/1 service acknowledged MSFS adapter` and
`MSFS telemetry stream active: 12 SimVars` messages.
