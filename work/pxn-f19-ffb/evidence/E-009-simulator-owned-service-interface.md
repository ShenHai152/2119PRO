# E-009 — Simulator-owned PXNF/1 motor service interface

- title: Vibration content is separated from PXN device output
- observed_at: 2026-08-03
- source_type: file
- source_ref: `ffb-bridge/service_windows.go`, `PXNF-PROTOCOL.md`, `adapters/`
- content_hash: n/a (delivery hashes are recorded in the report)
- repro_command: |
    Start `PXN2119FFBBridge.exe --mode service`, then run `send-vibration.ps1 -Amplitude 255 -Seconds 3` from a second PowerShell window.
- linked_workitem: WI-008
- supersedes: none

## Evidence

The default runtime is a localhost-only UDP motor service. A 16-byte `PXNF/1`
command contains final left/right amplitude, watchdog duration, and sequence.
The service applies the final amplitude without interpreting touchdown, stall,
engine, runway, turbulence, or other simulator concepts.

Every SET command has a maximum five-second lifetime. Missing refresh packets,
an explicit STOP command, or process shutdown sends a zero motor report.

Simulator adapter sources are split by platform:

- MSFS 2020/2024: SimConnect ClientData `PXN2119_FFB` to PXNF/1 UDP.
- X-Plane 12: writable `pxn/ffb/*` DataRefs to PXNF/1 UDP at 50 Hz.

The simulator SDKs are not present in the analysis workspace, so adapter source
is delivered for SDK builds but no adapter binary is claimed as validated.
