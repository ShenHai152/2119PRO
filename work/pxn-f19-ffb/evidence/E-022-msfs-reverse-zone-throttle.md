# E-022 — MSFS adapter owns a configurable PXN-F19 reverse-zone throttle axis

- title: MSFS adapter owns a configurable PXN-F19 reverse-zone throttle axis
- observed_at: 2026-08-03T16:11:14+08:00
- source_type: file | command
- source_ref: `uires.zip/xml/functionTestSetting2119.xml`; `E-012`; `ffb-bridge/internal/throttlezone`; `ffb-bridge/cmd/pxn-msfs-adapter`
- content_hash: `zone.go=36ee193f0159b9764d3157a3286dc1b3ee9159d8de0702ff8f71362dc54a1c5c`; `throttle_input_windows.go=9aa971824c779dd6a58335e936e46cc7b4351d4174a97be1432ad37a136ce194`; `reverse_throttle_windows.go=8b035b6706beeaa085905e978f89a447daa7017b53be64c5c6372fe46abe06ce`; `PXN2119FFBBridge.exe=32a31f5350122b83f85e29acedb6133f5199568e59fed314df37fff6b6f3588b`; `PXNMSFSAdapter.exe=1a1e223ab649aa32c86776cf4208e6f8d0d6f0950c6c61167a1d7d3b2f46ab4f`; `PXN2119-FFB-Bridge.zip=04282aa44a34f8e83d6069bec4f09b473ce07e2df56716494aa5a8099bf8a011`
- repro_command: |
    `strings -a /tmp/pxn-unpacked/app/PXNJoystick.exe | rg -i 'Throttle|SliderZ|2119'`
    `sed -n '1,80p' /tmp/pxn-uires-decrypted/xml/functionTestSetting2119.xml`
    `env GOCACHE=/tmp/pxn-reverse-zone-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -count=1 -cover -buildvcs=false ./...`
    `env GOCACHE=/tmp/pxn-reverse-zone-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go vet ./...`
    `env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOCACHE=/tmp/pxn-reverse-zone-win-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -c -buildvcs=false -o /tmp/PXNMSFSAdapter-reverse-zone.test.exe ./cmd/pxn-msfs-adapter`
- raw_excerpt: |
    The vendor function-test UI exposes `SliderZ` with range 0..255, and the
    recovered F3 table identifies slot 0x25 as `Z/throttle`. No recovered F3 or
    F4 setting expands that physical axis into a negative range.

    The optional adapter mode opens PXN-F19 through DirectInput in
    background/nonexclusive mode and requests only the Z-axis object, normalized
    to 0..65535. With defaults `reverse-zone=0.12`, `reverse-neutral=0.02`, and
    `idle-end=high`, the bottom 12% maps from -16383 to 0, the next 2% maps to
    idle, and the remaining travel maps from 0 to +16383. The signed result is
    transmitted through SimConnect `AXIS_THROTTLE_SET`.

    Unit tests cover both raw-axis directions, full reverse, both neutral
    boundaries, rescaled forward travel, disabled-mode compatibility, and
    invalid configuration. All Go packages, vet, and Windows amd64 cross-builds
    passed. Pause, SimStop, Quit, input loss, and adapter shutdown send zero.
- linked_workitem: WI-016
- supersedes: none

MSFS runtime behavior remains candidate until the user removes the original
physical throttle binding and validates the signed axis with real aircraft.
Aircraft using custom WASM throttle logic may require a model-specific event.
