# Timeline

## 2026-08-03T11:24:09+08:00 | lead | routing
- action: Routed the case to radare2 with protocol-reverse as a supporting analysis surface.
- command_or_ref: reverse-skill/MASTER-ROUTING.md R7
- result_summary: PRIMARY=radare2; lead_role=lead; specialist_roles=[cre, doc].
- artifacts: [scope.md, workitems.md]
- evidence_ids: []
- next: Perform clean PE triage and rebuild the DirectInput/HID call flow from evidence.

## 2026-08-03T11:33:42+08:00 | cre | device-path and effect-driver recovery
- action: Recovered EZFRD DeviceID path selection, COM registration, DownloadEffect update masks, and motor WriteFile path.
- command_or_ref: radare2 analysis of EZFRD64.dll functions 0x180001600, 0x180001cc0, 0x180001e60, 0x180002600, 0x180002a60
- result_summary: HID path is matched by DirectInput instance GUID; 0x80040154 is unregistered CLSID; bridge effect GUIDs and DIEP masks were wrong.
- artifacts: [evidence/E-001-device-path-selection.md, evidence/E-002-motor-output.md, evidence/E-003-directinput-effect-update.md, evidence/E-004-com-registration.md, findings.md]
- evidence_ids: [E-001, E-002, E-003, E-004]
- next: Correct, rebuild, package, and have the user run elevated registration followed by the hardware test.

## 2026-08-03T11:33:42+08:00 | lead | bridge correction
- action: Corrected DirectInput effect GUIDs and update flags, improved COM registration path verification, and rebuilt the Windows amd64 executable.
- command_or_ref: go test ./...; go build -buildvcs=false -trimpath -ldflags='-s -w'
- result_summary: Build passed; new executable SHA-256 is 1a9873976893d2a117ceccbbc463d4dd0ed27d46e9962ea7347c764b06b514a6.
- artifacts: [../../ffb-bridge/main_windows.go, ../../ffb-bridge/register-ezfrd.cmd, ../../ffb-bridge/PXN2119FFBBridge.exe]
- evidence_ids: [E-003, E-004]
- next: Refresh delivery archive and perform real-device validation on Windows.

## 2026-08-03T11:44:00+08:00 | cre | effect runtime analysis
- action: Mapped StartEffect, the 150 ms timer scan, and the force evaluator switch table.
- command_or_ref: radare2 analysis of EZFRD64.dll functions 0x180002210, 0x180002840, 0x180003210
- result_summary: ConstantForce effect ID 0 maps magnitude directly; Sine effect ID 3 is timer sampled and can alias.
- artifacts: [evidence/E-005-effect-sampling.md, findings.md]
- evidence_ids: [E-005]
- next: Prefer ConstantForce, expose SetParameters HRESULTs, rebuild and validate on hardware.

## 2026-08-03T12:20:00+08:00 | lead | DirectInput-selected HID bypass
- action: Replaced guessed SetupAPI/MI_00 selection with DIPROP_GUIDANDPATH and added opt-in native motor output.
- command_or_ref: go test -buildvcs=false ./...; PXN2119FFBBridge.exe --mode test --output hidpath
- result_summary: Source passes Windows amd64 cross-build tests; exact-path backend uses verified 00 05 01 AMP AMP 00 00 00 report at full byte amplitude. Real-device validation remains pending.
- artifacts: [evidence/E-006-directinput-selected-hid-bypass.md, ../../ffb-bridge/main_windows.go, ../../ffb-bridge/README.md]
- evidence_ids: [E-001, E-002, E-006]
- next: Have the user validate physical vibration with the hidpath backend.

## 2026-08-03T12:32:00+08:00 | doc | final build and report
- action: Cross-built the Windows amd64 bridge, rebuilt the delivery archive, verified the archive payload, and generated the evidence-linked Chinese report.
- command_or_ref: go test; go build; bsdtar; sha256sum
- result_summary: Initial hidpath EXE/ZIP build completed and archive payload matched.
- artifacts: [../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip, ../../2026-08-03_逆向-PXN-F19-FFB-report.md]
- evidence_ids: [E-006]
- next: Real-device validation using `PXN2119FFBBridge.exe --mode test --output hidpath`.

## 2026-08-03T12:45:00+08:00 | lead | hardware validation and strength tuning
- action: Recorded successful real-device vibration, extended the full-amplitude test pulse, and added clamped simulator gain.
- command_or_ref: PXN2119FFBBridge.exe --mode test --output hidpath; go test; go build
- result_summary: PXN-F19 physically vibrates through the DirectInput-selected HID path. Final build makes hidpath the default; EXE SHA-256 f3a044ce422e0b4bccb279f3c5947dbcd2b95112b5ee1621f28750e515b3d454; ZIP SHA-256 4e840b7767dafd4a87d17c2feb2b0e998d6e8018596b57e11101a0a485630043.
- artifacts: [evidence/E-007-hardware-vibration-validation.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-007]
- next: Validate and tune MSFS 2020/2024 and X-Plane 12 telemetry in flight.

## 2026-08-03T13:02:00+08:00 | lead | motor-stop lifecycle fix
- action: Added Go interrupt handling and a native SetConsoleCtrlHandler that waits for the zero-amplitude report before process termination.
- command_or_ref: go test; go build; bsdtar; sha256sum
- result_summary: Windows amd64 build passed. EXE SHA-256 eb6b2d8ad6dce20bab671d6a9548d272fba1e4c8aca6ebb83eeffcf5c76a5968; ZIP SHA-256 10c348eee3527ccebb06c4ccdfa33130f5b2d24f48db28725a09555ea948f433; archive EXE matches the build.
- artifacts: [evidence/E-008-motor-stop-on-process-exit.md, ../../ffb-bridge/main_windows.go, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-008]
- next: User validates that Ctrl+C and console-close both stop vibration immediately.

## 2026-08-03T13:40:00+08:00 | lead | simulator-owned service architecture
- action: Made a localhost PXNF/1 motor service the default, added a watchdog and SDK-facing protocol, and separated MSFS/X-Plane adapter sources from the device core.
- command_or_ref: go test; go build; bsdtar; sha256sum
- result_summary: Core Windows amd64 service build passed. The receive loop drains queued packets and applies the newest command. EXE SHA-256 7559aaf93f8002eae85d6390fbf68654782a6b41094f8f54687dfc9b5deb7246; ZIP SHA-256 928ebe7d733109797d591621259d021828090efaf94022b4f5a07468a8a01e32. Adapter binaries remain SDK-dependent.
- artifacts: [evidence/E-009-simulator-owned-service-interface.md, ../../ffb-bridge/service_windows.go, ../../ffb-bridge/PXNF-PROTOCOL.md, ../../ffb-bridge/adapters, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-009]
- next: Validate service commands on Windows, then build each adapter with the official simulator SDK.

## 2026-08-03T14:00:00+08:00 | lead | MSFS legacy-driver crash isolation
- action: Correlated the reported MSFS 2020 attached-device startup crash with EZFRD's in-process COM registration, raw HID initialization, and recurring timer; added a reversible isolation script and removed the FFB-only filter from native path discovery.
- command_or_ref: radare2 analysis of EZFRD64.dll exports and functions 0x1800011c0, 0x180001480, 0x1800018a0, 0x1800018b0, 0x180001cc0; go test; go build; bsdtar; sha256sum
- result_summary: Static mechanism validated and Windows amd64 build passed. EXE SHA-256 07358f8f43ce74e1a5588731769cad1bab4e0d88a815c2f2861cf94e532f6344; ZIP SHA-256 2dade070e3b06ce36db764f0cf974a323860545439523d8246f4ab6c56f94cda. The user then confirmed that MSFS 2020 no longer crashed with PXN-F19 attached after EZFRD isolation.
- artifacts: [evidence/E-010-msfs-ezfrd-inprocess-crash-isolation.md, ../../ffb-bridge/disable-ezfrd-for-games.cmd, ../../ffb-bridge/main_windows.go, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-010, E-014]
- next: Preserve a crash dump only if exact root-cause localization in the original DLL is required; separately test X-Plane 12 startup.

## 2026-08-03T15:45:00+08:00 | lead | hidden onboard feature recovery
- action: Decrypted the vendor UI resource archive and recovered the PXN2119 F1/F2/F3/F4 feature-report builders, upload loops, configuration record layout, and EZFRD effect evaluator matrix.
- command_or_ref: Static radare2 analysis of PXNJoystick.exe and EZFRD64.dll; offline extraction of uires.zip with the embedded resource password.
- result_summary: Confirmed three onboard modes, 24 active remapping slots, per-button rapid fire, keyboard/mouse/joystick mappings, ten axis mappings, seven dead zones, dual-motor preview, profile import/export, and seven implemented DirectInput effect families. Spring, Damper, Inertia, and Friction are stubs. No unknown report was sent to hardware.
- artifacts: [evidence/E-011-encrypted-ui-and-onboard-features.md, evidence/E-012-onboard-feature-report-protocol.md, evidence/E-013-ezfrd-effect-matrix.md, evidence/E-014-msfs-isolation-hardware-validation.md, findings.md, ../../2026-08-03_逆向-PXN-F19-FFB-report.md]
- evidence_ids: [E-011, E-012, E-013, E-014]
- next: Treat reserved F4 IDs and mouse buttons 6-8 as candidates unless a safe backed-up hardware test is later authorized.

## 2026-08-03T17:10:00+08:00 | lead | dual motor calibration and reconnect delivery
- action: Replaced PXNF/1 channel collapsing with independent joystick-base/throttle native HID output, added versioned per-user calibration, an interactive calibration wizard, one-shot channel tests, and service-mode automatic reconnect without stale-command replay. Legacy left/right names remain aliases.
- command_or_ref: go test; go vet; Windows test-binary cross-compile; Windows release build; archive extraction and hash comparison
- result_summary: Portable unit tests passed. Windows amd64 test binary and release executable compiled. EXE SHA-256 ae9331240c910de4294bf00cac90d81435eaf4341d17b85335584f8854844408; ZIP SHA-256 aa45878dfe597b4351ab34bb74fb1b95427908424e0e81f8296e3945d50ea7e8. Hardware validation of distinct joystick/throttle actuation remains pending.
- artifacts: [evidence/E-015-independent-motor-calibration-and-reconnect.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip, ../../ffb-bridge/README.md, ../../ffb-bridge/PXNF-PROTOCOL.md]
- evidence_ids: [E-015]
- next: Run stick-only, throttle-only, both-channel, calibration, unplug/replug, watchdog, and MSFS startup acceptance tests on Windows.

## 2026-08-03T18:05:00+08:00 | lead | simulator identity and runtime status delivery
- action: Added adapter identity and simulator lifecycle flags to the reserved PXNF/1 field, periodic service status, idle HID health checks, adapter timeout tracking, and immediate two-channel stop behavior for simulator pause, stop, exit, plugin unload, and device loss.
- command_or_ref: go test -count=1 -cover; go vet; Windows amd64 test-binary cross-compile; Windows release build; archive extraction and hash comparison
- result_summary: Go tests passed with 61.7% statement coverage. Windows amd64 test and release binaries compiled. The service reports MSFS 2020, MSFS 2024, X-Plane 12, or generic MSFS; calibrated stick/throttle amplitudes; device disconnect/reconnect; and watchdog-stopped state. EXE SHA-256 ff5dbac6974c77bacfb55a8a9430a199d988e48bd75cfb79c622f45e231d725d; ZIP SHA-256 8aa52d37f32bd1b6fc2f5d6503c00f72cf7161bed6aeb558828fc431e9426a17.
- artifacts: [evidence/E-016-simulator-identity-and-runtime-status.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip, ../../ffb-bridge/PXNF-PROTOCOL.md, ../../ffb-bridge/adapters]
- evidence_ids: [E-016]
- next: Compile the adapters with the official SimConnect and X-Plane SDKs, then acceptance-test pause, resume, exit, unplug/replug, and watchdog behavior in all three simulators.

## 2026-08-03T19:10:00+08:00 | lead | compiled MSFS adapter and automatic lifecycle supervision
- action: Reimplemented the MSFS adapter as a Windows amd64 Go executable with dynamic SimConnect loading, added an offline version/lifecycle self-test, and made service mode monitor MSFS processes and automatically start/reap the adjacent adapter.
- command_or_ref: go run adapter --self-test; go test -count=1 -cover; go vet; Windows amd64 release builds; ZIP payload hash comparison
- result_summary: The self-test validated MSFS 2020/2024 OPEN detection plus Pause, SimStop, and Quit STOP actions. All Go packages passed tests; both Windows executables compiled. A non-blocking ACK drain protects long-running UDP sessions. Bridge SHA-256 62a8c54780e7d53da170e5d1652496414c703366017191ea47c711eef4ccd6b4; adapter SHA-256 08c60d2255e68360b6f01c046e8b64282301df0e9e7e2892590e1814046aa9fe; ZIP SHA-256 4eafc27ba6fde1a3e30b6fc374de8c55eea472e68f96fa7b9322c9fcaa8e377f. Archive EXEs match the standalone builds.
- artifacts: [evidence/E-017-compiled-msfs-adapter-and-supervisor.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-017]
- next: Place SimConnect.dll beside the adapter if Windows cannot resolve it, then acceptance-test automatic start, 2020/2024 identity, Pause, SimStop, Quit, and simulator-crash fallback on Windows.

## 2026-08-03T19:35:00+08:00 | lead | MSFS 2020 KittyHawk runtime identity correction
- action: Recorded a real MSFS 2020 SimConnect session, identified `KittyHawk` version 11.0 as the OPEN identity, added the explicit MSFS 2020 mapping, rebuilt the adapter, and refreshed the release archive.
- command_or_ref: user runtime log; go test; adapter self-test; Windows amd64 adapter build; ZIP payload hash comparison
- result_summary: Real MSFS 2020 delivered Pause, SimStop, SimStart, and resume events. The corrected adapter now reports MSFS 2020 instead of generic Microsoft Flight Simulator. Adapter SHA-256 d39aa9421f60bf6f6ecd34649b9a0cc96c42e3fb081356ad601b5222d49e9b6a; ZIP SHA-256 8a2edb3c7048c3fe6a90fc081e34f5e110c260e94d0a197a6614090fdf77d74c.
- artifacts: [evidence/E-018-msfs2020-kittyhawk-runtime-events.md, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-018]
- next: Replace only PXNMSFSAdapter.exe, confirm bridge status changes to adapter=MSFS 2020, then test automatic launch and Quit.

## 2026-08-03T20:05:00+08:00 | lead | MSFS SimVar vibration generation
- action: Corrected the no-vibration architecture by subscribing the adapter directly to MSFS acceleration, G-force, stall, ground, velocity, RPM, N1, and combustion SimVars; added an independent smoothed stick/throttle mixer and automatic gain forwarding from the bridge.
- command_or_ref: go test -count=1 -cover; go vet; Windows test-binary cross-compile; Windows bridge/adapter release builds; ZIP payload hash comparison
- result_summary: The adapter no longer requires a separate aircraft/WASM ClientData producer for basic vibration. Engine data drives the throttle motor; airframe, stall, touchdown, and ground-roll data drive the joystick-base motor. External ClientData commands retain priority. Bridge SHA-256 aad5e718e84b3c057f0afa30c103dbf4e64ad670596542e1913baa2961666aeb; adapter SHA-256 2872a89527d52e4497cb41476abb738597ebe20fc6921f6ef93f05fff2b91a84; ZIP SHA-256 4a705de46009cdc14e77affc2b461513053167402b761afeeacd7ff6ff2b9637.
- artifacts: [evidence/E-019-msfs-simvar-vibration-mixer.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-019]
- next: Replace both EXEs, run MSFS 2020 in flight, confirm non-zero telemetry status, pause STOP, and tune `--msfs-telemetry-gain` if required.

## 2026-08-03T20:30:00+08:00 | lead | Windows UDP polling and SimStart resume correction
- action: Correlated real logs showing adapter events but bridge adapter=none with an already-expired UDP read deadline; changed it to a short future deadline, made SimStart clear latched Pause, relaxed telemetry DefineCount handling, and added ACK/telemetry-stream diagnostics.
- command_or_ref: user runtime log; go test -count=1 -cover; go vet; Windows test/release cross-build; ZIP payload verification
- result_summary: New builds report whether the service acknowledges PXNF/1 and whether twelve SimVars are flowing. Bridge SHA-256 1e65ecb6a92d371eacfb9686cc6c84349b4610777c0c0ea34c6e1d61e8f8f246; adapter SHA-256 9028829020a0e7563b55df5c38c402e2be6c23ff81afaf324253b7b58b1424d7; ZIP SHA-256 19543eccd63a6bd09f6ea9630ef6935032f3435936af806d6f2ea98af639f90f.
- artifacts: [evidence/E-020-windows-udp-poll-and-simstart-resume-fix.md, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-020]
- next: Replace both EXEs and confirm service ACK, telemetry stream activation, non-zero amplitudes, physical vibration, and pause STOP.

## 2026-08-03T15:49:01+08:00 | lead | final motor safety envelope
- action: Correlated the user's gain-2.0 disconnect and gain-1.5 stability result with the unbounded post-calibration HID path; added final per-channel caps, proportional shared power budget, rising slew, rate-limited diagnostics, and actual-applied status reporting while preserving immediate STOP.
- command_or_ref: user hardware report; go test -count=1 -cover; go vet; Windows amd64 test-binary cross-compile; Windows release builds; ZIP payload verification
- result_summary: All Go packages passed with bridge coverage 61.0%; vet and Windows cross-builds passed. Bridge SHA-256 b1801c9b649164b7575b2e6ec8c988bad282d9aafd481742bbf27e63e7a37b86; adapter SHA-256 9028829020a0e7563b55df5c38c402e2be6c23ff81afaf324253b7b58b1424d7; ZIP SHA-256 3d9477165e89944766297d8ec34c35225578ae2003a6c8966b7281a7f80fb2c6.
- artifacts: [evidence/E-021-final-motor-safety-envelope.md, ../../ffb-bridge/motor_safety.go, ../../ffb-bridge/motor_safety_test.go, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-021]
- next: Replace the bridge EXE, keep telemetry gain at 1.5, run a sustained MSFS flight, and confirm no device disconnect plus immediate Pause/Exit stop.

## 2026-08-03T16:11:14+08:00 | lead | optional MSFS reverse-zone throttle
- action: Confirmed the vendor UI and recovered F3 protocol identify the physical oil-throttle input as Z/throttle without a negative-range firmware option; added optional DirectInput Z-axis ownership, a configurable 12% reverse and 2% neutral curve, signed SimConnect AXIS_THROTTLE_SET output, direction selection, reconnect, status logging, and immediate idle on lifecycle stop.
- command_or_ref: strings/XML inspection; E-012; go test -count=1 -cover; go vet; Windows amd64 test-binary cross-compile; Windows release builds; ZIP payload verification
- result_summary: All Go packages passed; throttle-zone coverage is 81.1%. Bridge SHA-256 32a31f5350122b83f85e29acedb6133f5199568e59fed314df37fff6b6f3588b; adapter SHA-256 1a1e223ab649aa32c86776cf4208e6f8d0d6f0950c6c61167a1d7d3b2f46ab4f; ZIP SHA-256 04282aa44a34f8e83d6069bec4f09b473ce07e2df56716494aa5a8099bf8a011.
- artifacts: [evidence/E-022-msfs-reverse-zone-throttle.md, ../../ffb-bridge/internal/throttlezone, ../../ffb-bridge/cmd/pxn-msfs-adapter/throttle_input_windows.go, ../../ffb-bridge/cmd/pxn-msfs-adapter/reverse_throttle_windows.go, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-012, E-022]
- next: Remove the PXN-F19 throttle axis from MSFS Controls, launch with --msfs-reverse-zone 0.12, confirm raw direction and idle band, then validate reverse on a stock aircraft before adding aircraft-specific profiles.

## 2026-08-03T16:32:24+08:00 | lead | explicit MSFS reverser engagement
- action: Used the user's real log to validate raw Z-axis direction and full negative output while disproving signed-generic-axis-only reverse engagement; added generic plus engine 1–4 axis transmission, repeated reverse Hold, transition-safe Toggle fallback, shutdown release, diagnostics, and state-machine tests.
- command_or_ref: user MSFS 2020 runtime log; go test -count=1 -cover; go vet; Windows amd64 test-binary cross-compile; Windows release builds; ZIP payload verification
- result_summary: Adapter tests now cover 74.1% of statements. Bridge SHA-256 dc1b3c80c98f847cb304c081927adab7ef812e58972e253bbde675edc4be4bc5; adapter SHA-256 4821c4aafd47fec3bdd5bd794ebfcf902e97c85dbddf96555c1e6e54abd0d6ce; ZIP SHA-256 3d8c26c1e09f97689ec12e80803aa171886527eb7fc28d0b0bbf54a0da6adecd.
- artifacts: [evidence/E-023-msfs-negative-axis-requires-reverse-event.md, ../../ffb-bridge/cmd/pxn-msfs-adapter/reverse_mode.go, ../../ffb-bridge/cmd/pxn-msfs-adapter/reverse_throttle_windows.go, ../../ffb-bridge/PXN2119FFBBridge.exe, ../../ffb-bridge/PXNMSFSAdapter.exe, ../../PXN2119-FFB-Bridge.zip]
- evidence_ids: [E-023]
- next: Replace both EXEs and retest default Hold mode; if the aircraft still does not deploy reverse, run the same test with --msfs-reverse-event toggle and report the aircraft model plus event log.
