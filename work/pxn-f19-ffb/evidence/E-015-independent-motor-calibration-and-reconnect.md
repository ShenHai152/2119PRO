# E-015 — Independent motor output, calibration, and reconnect implementation

- title: Independent motor output, calibration, and reconnect implementation
- observed_at: 2026-08-03T17:10:00+08:00
- source_type: command
- source_ref: `/tmp/pxn-unpacked/app/PXNJoystick.exe:0x004b1cc0,0x004b1d90,0x004b8a40`; `ffb-bridge/`
- content_hash: `PXNJoystick.exe=d759147b653e9fe917d811b664acaaff9286e9d78148e9bf8b60eb048817642a`; `PXN2119FFBBridge.exe=ae9331240c910de4294bf00cac90d81435eaf4341d17b85335584f8854844408`; `PXN2119-FFB-Bridge.zip=aa45878dfe597b4351ab34bb74fb1b95427908424e0e81f8296e3945d50ea7e8`
- repro_command: |
    `go test -buildvcs=false ./...`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go test -c -buildvcs=false -o /tmp/pxn2119-ffb-bridge.test.exe .`
    `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -buildvcs=false -trimpath -ldflags='-s -w' -o PXN2119FFBBridge.exe .`
    `sha256sum PXN2119FFBBridge.exe ../PXN2119-FFB-Bridge.zip`
- raw_excerpt: |
    PXNJoystick's two motor preview handlers scale the joystick-base and
    throttle strengths independently. The worker writes
    `00 05 01 STICK THROTTLE 00 00 00`; these are the fields historically
    named `left/right` in PXNF/1.
    The bridge now preserves the two PXNF/1 bytes through service parsing,
    calibration, and native HID output. Portable tests validate independent
    channels, calibration bounds, watchdog stop, atomic config round-trip,
    reconnect-zero behavior, and no replay after a failed output. Linux tests,
    Go vet, Windows test-binary compilation, Windows executable build, and
    archive payload hash verification completed successfully.
- linked_workitem: WI-011
- supersedes: none

Independent physical joystick/throttle actuation still requires a Windows
hardware test. No unknown HID feature report was sent during implementation or
testing.
