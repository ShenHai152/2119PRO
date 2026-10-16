# E-021 — Final motor safety envelope prevents sustained overload requests

- title: Final motor safety envelope prevents sustained overload requests
- observed_at: 2026-08-03T15:49:01+08:00
- source_type: manual | command | file
- source_ref: User MSFS hardware result; `ffb-bridge/motor_safety.go`; `ffb-bridge/motor_safety_test.go`
- content_hash: `motor_safety.go=2ee5e41d8144561a0cc2c97dcf7fe499c792c6ed87963a14fac7f6efae811522`; `motor_safety_test.go=9ed2fc3e7e5165cbf5896d3bcf3ce7dc6478ccea03877cdf82363d00a9373822`; `PXN2119FFBBridge.exe=b1801c9b649164b7575b2e6ec8c988bad282d9aafd481742bbf27e63e7a37b86`; `PXNMSFSAdapter.exe=9028829020a0e7563b55df5c38c402e2be6c23ff81afaf324253b7b58b1424d7`; `PXN2119-FFB-Bridge.zip=3d9477165e89944766297d8ec34c35225578ae2003a6c8966b7281a7f80fb2c6`
- repro_command: |
    `env GOCACHE=/tmp/pxn-safety-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -count=1 -cover -buildvcs=false ./...`
    `env GOCACHE=/tmp/pxn-safety-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go vet ./...`
    `env GOOS=windows GOARCH=amd64 CGO_ENABLED=0 GOCACHE=/tmp/pxn-safety-win-gocache GOMODCACHE=/tmp/pxn-go-modcache /tmp/go-toolchain/bin/go test -c -buildvcs=false -o /tmp/PXN2119FFBBridge-safety.test.exe .`
    `sha256sum PXN2119FFBBridge.exe PXNMSFSAdapter.exe ../PXN2119-FFB-Bridge.zip`
- raw_excerpt: |
    User hardware result: `--msfs-telemetry-gain 2` vibrated temporarily and
    then the PXN-F19 disconnected; reducing it to `1.5` remained stable. The
    two physical motors also had different subjective strength.

    The revised service applies safety after saved per-channel calibration and
    before the native HID write. Defaults are `stick-max=220`,
    `throttle-max=220`, combined `motor-budget=330`, and a rising slew of 24
    byte units per output update. Proportional budget scaling preserves the
    simulator-requested channel balance. Every falling edge is immediate, so a
    STOP, Pause, SimStop, Quit, watchdog expiry, or adapter disconnect reaches
    `stick=0 throttle=0` without a decay ramp.

    Unit tests cover per-channel caps, proportional shared-budget scaling,
    rising slew, immediate STOP, eventual target convergence, and preservation
    of a low non-zero channel. All Go packages passed tests and vet; Windows
    amd64 test binaries and both release executables compiled successfully.
- linked_workitem: WI-015
- supersedes: none

The overload cause remains a high-confidence hardware-protection hypothesis,
not a measured USB current trace. Real-device validation of the new safety
defaults is still required; no unknown HID Feature Report was sent.
