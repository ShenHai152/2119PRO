# E-008 — Motor stop was skipped on forced default process termination

- title: Windows process termination bypassed the deferred zero-amplitude report
- observed_at: 2026-08-03
- source_type: manual
- source_ref: user-operated PXN-F19 and `ffb-bridge/main_windows.go`
- content_hash: n/a
- repro_command: |
    Start `PXN2119FFBBridge.exe --mode test`, terminate the old build, and observe that the last motor amplitude remains latched.
- raw_excerpt: |
    User result: vibration did not stop after the program ended.
- linked_workitem: WI-007
- supersedes: none

## Finding

Go's default Windows Ctrl+C termination path does not run application defers.
The old build therefore skipped `hidMotor.close()` and never sent
`00 05 01 00 00 00 00 00`. The device retained its last commanded amplitude.

## Fix

The bridge now catches `os.Interrupt` and also registers
`SetConsoleCtrlHandler` for Ctrl+C, Ctrl+Break, console close, logoff, and
shutdown. The native handler waits until main returns and `output.close()` has
synchronously sent the zero-amplitude report.
