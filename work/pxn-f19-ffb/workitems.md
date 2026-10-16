# Work Items

| ID | title | role | targets | surface | status | evidence | notes |
|----|-------|------|---------|---------|--------|----------|-------|
| WI-001 | Clean PE triage | cre | PXNJoystick.exe, EZFRD*.dll | binary | complete | E-001,E-002,E-004 | Hashes, imports, exports, RTTI and key functions recovered |
| WI-002 | Recover DirectInput effect-driver registration | cre | EZFRD*.dll | COM/DirectInput | complete | E-003,E-004 | CLSID, vtable, DownloadEffect masks and registration recovered |
| WI-003 | Recover PXN-F19 output path and packet semantics | cre | PXNJoystick.exe, EZFRD*.dll | HID | complete | E-001,E-002 | Instance-GUID path matching and motor report validated statically |
| WI-004 | Review and correct bridge architecture | lead | ffb-bridge | implementation | complete | E-003,E-004,E-005 | Corrected GUIDs/DIEP flags; prefer ConstantForce; kept unsafe native HID disabled |
| WI-005 | Generate evidence-linked report | doc | case artifacts | report | complete | E-001,E-002,E-003,E-004,E-005,E-006 | Formal Chinese report includes Evidence, Findings, Path, timeline, and hashes |
| WI-006 | Implement DirectInput-selected native HID backend | lead | ffb-bridge | DirectInput/HID | complete | E-001,E-002,E-006,E-007 | Real-device vibration validated; strength tuning added |
| WI-007 | Stop latched motor output on Windows termination | lead | ffb-bridge | process lifecycle/HID | complete | E-008 | Signal and native console-control cleanup added; Windows validation pending |
| WI-008 | Separate simulator effect content from motor service | lead | ffb-bridge/adapters | UDP/SimConnect/XPLM | complete | E-009 | Core service built; adapter sources require official simulator SDKs |
| WI-009 | Isolate legacy EZFRD from simulator processes | lead | EZFRD64.dll, ffb-bridge | COM/DirectInput/HID | complete | E-010,E-014 | Reversible 32/64-bit unregister script built; user confirmed MSFS 2020 no longer crashes with PXN-F19 attached |
| WI-010 | Recover hidden onboard configuration and effect matrix | lead | PXNJoystick.exe, uires.zip, EZFRD64.dll | HID/UI/DirectInput | complete | E-011,E-012,E-013 | Three modes, mappings, rapid fire, dead zones, profile handling, motor preview, and effect coverage recovered without unknown hardware probes |
| WI-011 | Preserve stick/throttle motor channels and add calibration/reconnect | lead | ffb-bridge | Go/HID/UDP | implementation_complete | E-015 | Portable tests and Windows cross-build passed; independent joystick/throttle hardware validation pending |
| WI-012 | Add simulator identity, runtime status, and pause/exit motor stop | lead | ffb-bridge/adapters | Go/UDP/SimConnect/XPLM | implementation_complete | E-016 | Go tests, vet, Windows cross-build, release build, and package verification passed; official SDK builds and simulator runtime acceptance pending |
| WI-013 | Compile and supervise the MSFS 2020/2024 adapter | lead | ffb-bridge/cmd/pxn-msfs-adapter | Go/Win32/SimConnect/UDP | runtime_partial | E-017,E-018 | MSFS 2020 real OPEN/Pause/SimStop/SimStart/resume events validated; KittyHawk mapping fixed; automatic child launch, bridge status, Quit, and MSFS 2024 acceptance pending |
| WI-014 | Generate MSFS vibration from SimVars | lead | ffb-bridge/cmd/pxn-msfs-adapter | SimConnect/telemetry/PXNF | implementation_complete | E-019,E-020 | Independent mixer plus Windows UDP deadline, SimStart resume, ACK, and telemetry diagnostics completed; real-flight acceptance pending |
| WI-015 | Protect PXN-F19 from sustained dual-motor overload | lead | ffb-bridge | Go/HID/PXNF | implementation_complete | E-021 | Final calibrated bytes now pass per-channel caps, a proportional shared budget, and rising slew; immediate STOP is preserved; Windows hardware retest pending |
| WI-016 | Map the bottom PXN throttle travel to MSFS reverse thrust | lead | ffb-bridge/cmd/pxn-msfs-adapter | DirectInput/SimConnect | implementation_complete | E-012,E-022 | Optional 12% reverse + 2% neutral signed axis implemented; original MSFS binding must be removed; real-aircraft acceptance pending |
| WI-017 | Explicitly unlock MSFS reverse thrust for negative-axis aircraft | lead | ffb-bridge/cmd/pxn-msfs-adapter | SimConnect events | implementation_complete | E-023 | Real log validated axis/curve but showed negative generic axis clamped to idle; Hold plus engine 1–4 axes added, Toggle fallback available; retest pending |

## Coverage
- [x] Triage complete for in_scope assets
- [x] DirectInput/COM findings validated
- [x] HID packet semantics validated
- [x] Findings have Evidence
- [x] Callflow Path documented
- [x] Timeline started
- [x] Report exported as Markdown case artifact
- [x] MSFS crash-isolation mechanism documented
- [x] MSFS 2020 isolation validated on Windows hardware
- [ ] X-Plane 12 startup behavior tested
- [x] Hidden onboard feature surface documented
- [x] Independent channel implementation and calibration tests complete
- [x] Simulator/device lifecycle status implementation complete
- [x] Pause, stop, exit, timeout, and disconnect clear both logical motor channels
- [ ] Independent joystick/throttle motor output validated on Windows hardware
- [ ] X-Plane 12 adapter compiled with the official SDK and runtime-tested
- [x] MSFS 2020/2024 adapter Windows EXE cross-compiled without static SDK linkage
- [x] MSFS process detection and adapter automatic start/exit supervision implemented
- [ ] MSFS 2020 and MSFS 2024 real SimConnect lifecycle acceptance completed
- [x] MSFS SimVar telemetry produces independent stick/throttle PXNF amplitudes
- [x] Final HID output applies per-channel caps, a shared motor budget, and rising slew
- [ ] New motor safety defaults validated during a long MSFS hardware session
- [x] Optional MSFS reverse-zone throttle implementation and cross-build complete
- [ ] MSFS reverse-zone throttle validated after removing the physical axis binding
- [x] Real MSFS log validates PXN Z-axis direction and full negative output
- [ ] Explicit reverse Hold/Toggle event correction validated in MSFS
- [ ] MSFS 2020 real-flight amplitude and subjective tuning accepted
- [ ] field-journal written (anonymized)
