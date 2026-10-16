# E-006 — DirectInput-selected native HID bypass

- title: DirectInput can safely provide the exact PXN-F19 HID collection
- observed_at: 2026-08-03
- source_type: file
- source_ref: `ffb-bridge/main_windows.go`; `EZFRD64.dll:fcn.180001600`; `EZFRD64.dll:fcn.180002a60`
- content_hash: n/a (source and binary hashes are recorded in the final report)
- repro_command: |
    `PXN2119FFBBridge.exe --mode test --output hidpath`
- linked_workitem: WI-006
- supersedes: none

## Raw evidence

`EZFRD64.dll:fcn.180001600` obtains the configured DirectInput instance GUID,
enumerates candidate HID paths, calls DirectInput `FindDevice(candidatePath)`,
and accepts only the candidate whose returned instance GUID matches.

The bridge implements the equivalent identity binding through documented
DirectInput property `DIPROP_GUIDANDPATH` (`MAKEDIPROP(12)`). It validates that
the returned path contains `VID_11FF` and `PID_0847`, releases the DirectInput
objects, opens that exact path with read/write sharing, and initializes it with
the verified zero-amplitude report.

The output report recovered from `EZFRD64.dll:fcn.180002a60` is:

```text
00 05 01 AMP AMP 00 00 00
```

The driver duplicates the calculated maximum amplitude into both motor bytes.
The bridge therefore maps normalized amplitude `0..1` to byte `0..255` and does
not synthesize independent left/right values.

## Finding

This route preserves the vendor's instance-identity selection while bypassing
the EZFRD effect timer that accepted ConstantForce but produced no physical
motor response on the user's system. It is materially different from the
unsafe earlier fallback that guessed `MI_00`.
