# E-007 — PXN-F19 real-device vibration validation

- title: DirectInput-selected HID path drives the PXN-F19 vibration motor
- observed_at: 2026-08-03
- source_type: manual
- source_ref: user-operated PXN-F19 VID_11FF/PID_0847
- content_hash: n/a
- repro_command: |
    `PXN2119FFBBridge.exe --mode test --output hidpath`
- raw_excerpt: |
    User result: vibration occurred, but the short pulse was not obvious.
- linked_workitem: WI-006
- supersedes: none

## Finding

The exact HID path returned through DirectInput and the recovered
`00 05 01 AMP AMP 00 00 00` report produce physical vibration on the user's
PXN-F19. The remaining issue is perceived strength and motor spin-up time, not
device selection or packet validity.
