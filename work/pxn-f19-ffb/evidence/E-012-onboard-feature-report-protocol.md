# E-012 — PXN2119 onboard configuration feature-report protocol

- title: PXN2119 onboard configuration feature-report protocol
- observed_at: 2026-08-03T15:25:00+08:00
- source_type: command
- source_ref: `/tmp/pxn-unpacked/app/PXNJoystick.exe`
- content_hash: `d759147b653e9fe917d811b664acaaff9286e9d78148e9bf8b60eb048817642a`
- repro_command: |
    `radare2 -A -q -e scr.color=false -c 's 0x4b6bd0; pdf; s 0x4b6c10; pdf; s 0x4b6c50; pdf; s 0x4b6c90; pdf; s 0x4b9e00; pdf; s 0x4b9f70; pdf; s 0x4ba9d0; pdf; s 0x4baa70; pdf; q' /tmp/pxn-unpacked/app/PXNJoystick.exe`
- raw_excerpt: |
    Four wrappers call `HidD_SetFeature(handle, report, 0x40)` after setting
    report byte 1 to F1, F2, F3, or F4. Recovered 64-byte report prefixes:

    - F1: `00 F1 MODE SLOT MAP_TYPE MAP_VALUE ...`
    - F2: `00 F2 MODE SLOT FIRE_VALUE ...`
    - F3: `00 F3 MODE AXIS_SLOT MAP_TYPE MAP_VALUE ...`
    - F4: `00 F4 MODE PARAM_ID VALUE ...`

    F1 mapping types are 0=joystick, 1=keyboard, 2=mouse. F3 accepts types
    0..3 and uses the same keyboard/mouse renderers, with type 0 retaining the
    original axis behavior. Uploads cover modes 0, 1, and 2.

    F3 axis slots are:
    `21=X-`, `22=X+`, `23=Y-`, `24=Y+`, `25=Z/throttle`,
    `26=RZ-`, `27=RZ+`, `28=RX`, `29=RY`, `2A=Slider`.

    F4 dead-zone parameters are:
    `21=X`, `23=Y`, `25=Z`, `26=RY`, `28=RZ`, `29=RX`, `2A=Slider`,
    each constrained to 0..100.
- linked_workitem: WI-010
- supersedes: none

No unknown feature report was sent to hardware. The protocol was recovered from
packet construction, validation, UI handlers, and per-mode upload loops only.
