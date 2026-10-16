# E-014 — MSFS isolation validated on the user's PXN-F19

- title: MSFS isolation validated on the user's PXN-F19
- observed_at: 2026-08-03T14:20:00+08:00
- source_type: manual
- source_ref: user-provided Windows runtime result
- content_hash: n/a
- repro_command: |
    Run `disable-ezfrd-for-games.cmd`, keep PXN-F19 attached, start
    `PXN2119FFBBridge.exe --mode service`, and launch MSFS 2020.
- raw_excerpt: |
    User confirmation: `不会闪退了` / `不闪退了`.
    The user later confirmed that native HID vibration also worked again.
- linked_workitem: WI-009
- supersedes: none

This validates the isolation result on the reported machine. It does not prove
the exact exception instruction inside EZFRD64.dll because no crash dump was
available.
