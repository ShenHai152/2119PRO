# Case Scope

## meta
- case_id: 20260803-pxn-f19-ffb
- created: 2026-08-03T11:24:09+08:00
- operator: local
- primary_skill: radare2
- lead_role: lead
- specialist_roles: [cre, doc]

## auth
- status: granted
- basis: own_system
- evidence_of_auth: User supplied the installer and explicitly requested reverse engineering for their PXN-F19 hardware.

## in_scope
- assets:
  - /home/bearcandy/Project/2119PRO/PXNJoystick setupV3.0.exe
  - /tmp/pxn-unpacked/app/PXNJoystick.exe
  - /tmp/pxn-unpacked/app/EZFRD.dll
  - /tmp/pxn-unpacked/app/EZFRD64.dll
  - User-operated PXN-F19 VID_11FF/PID_0847
- surfaces: [windows_pe, directinput, com, hid]
- activities: [triage, static_reverse, protocol_reverse, implementation_review, report]

## out_of_scope
- assets: [unrelated USB devices, external production systems]
- activities: [firmware flashing, destructive USB writes, unsupported HID command fuzzing]

## network_profile
- mode: offline
- notes: Static analysis of local binaries only. Hardware validation is performed by the user on their own device.

## deliverables
- report: true
- field_journal: true
- diagrams: true
- timeline: true

## constraints
- timebox: iterative
- stealth: low
- data_handling: no_user_pii

## signoff
- ready_for_act: true
- checklist:
  - [x] auth.status = granted
  - [x] in_scope assets non-empty
  - [x] network_profile.mode chosen
  - [x] out_of_scope reviewed
