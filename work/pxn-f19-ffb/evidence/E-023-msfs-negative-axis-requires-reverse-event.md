# E-023 — MSFS receives the negative axis but does not unlock reverse thrust

- title: MSFS receives the negative axis but does not unlock reverse thrust
- observed_at: 2026-08-03T16:32:24+08:00
- source_type: log | file | command
- source_ref: User MSFS 2020 runtime log; `ffb-bridge/cmd/pxn-msfs-adapter/reverse_mode.go`; `reverse_throttle_windows.go`
- content_hash: `reverse_mode.go=efdb0c372f530f8933219547aa7f0e198a87bdc9bfa1f3f8f5fdc76748e1d24b`; `reverse_throttle_windows.go=1e9803f7e4205342bff33383f0c2e11b9ee872501b836897f254089cb7ed7123`; `PXN2119FFBBridge.exe=dc1b3c80c98f847cb304c081927adab7ef812e58972e253bbde675edc4be4bc5`; `PXNMSFSAdapter.exe=4821c4aafd47fec3bdd5bd794ebfcf902e97c85dbddf96555c1e6e54abd0d6ce`; `PXN2119-FFB-Bridge.zip=3d8c26c1e09f97689ec12e80803aa171886527eb7fc28d0b0bbf54a0da6adecd`
- repro_command: |
    `PXN2119FFBBridge.exe --mode service --msfs-telemetry-gain 1.5 --msfs-reverse-zone 0.12`
    `env GOCACHE=/tmp/pxn-reverse-hold-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -count=1 -cover -buildvcs=false ./...`
    `env GOCACHE=/tmp/pxn-reverse-hold-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go vet ./...`
    `env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOCACHE=/tmp/pxn-reverse-hold-win-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -c -buildvcs=false -o /tmp/PXNMSFSAdapter-reverse-hold.test.exe ./cmd/pxn-msfs-adapter`
- raw_excerpt: |
    The user's real MSFS 2020 log repeatedly showed:
    `MSFS throttle axis: raw=65535 position=0.0% output=-16383 mode=reverse`.
    This validates the PXN-F19 DirectInput Z-axis selection, high-end idle
    direction, and 12% reverse-zone calculation. The aircraft nevertheless did
    not enter reverse thrust; N1 returned toward idle instead.

    The revised adapter no longer relies on the generic signed axis alone. It
    maps and writes `AXIS_THROTTLE_SET` plus `AXIS_THROTTLE1_SET` through
    `AXIS_THROTTLE4_SET`. In the default `hold` mode it also repeats
    `THROTTLE_REVERSE_THRUST_HOLD` while the lever remains in the reverse zone.
    An optional `toggle` state machine sends
    `THROTTLE_REVERSE_THRUST_TOGGLE` only on reverse-zone transitions and
    returns it to normal on STOP. `axis` mode preserves the old behavior for
    diagnostics.
- linked_workitem: WI-017
- supersedes: none

The input curve is hardware-validated, but the explicit Hold/Toggle correction
still requires a second MSFS aircraft test.
