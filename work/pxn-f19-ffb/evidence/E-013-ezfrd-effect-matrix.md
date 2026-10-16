# E-013 — EZFRD effect implementation matrix

- title: EZFRD effect implementation matrix
- observed_at: 2026-08-03T15:40:00+08:00
- source_type: command
- source_ref: `/tmp/pxn-unpacked/app/EZFRD64.dll:0x180003210`
- content_hash: `a36571f371c6c0b1965aa1f0562b3356ddd44bd92566cbd188d17b68af0fa7e8`
- repro_command: |
    `radare2 -A -q -e scr.color=false -c 's 0x180003210; af; pdf; s 0x1800034e4; pxw 44; q' /tmp/pxn-unpacked/app/EZFRD64.dll`
- raw_excerpt: |
    The evaluator dispatches 11 DirectInput standard effect IDs through a jump
    table. IDs 0..6 have executable paths; IDs 7..10 jump directly to the zero
    result. The standard GUID order yields:

    - 0 ConstantForce: implemented
    - 1 RampForce: implemented
    - 2 Square: implemented through the shared periodic path
    - 3 Sine: implemented
    - 4 Triangle: implemented through the shared periodic path
    - 5 SawtoothUp: implemented through the shared periodic path
    - 6 SawtoothDown: implemented through the shared periodic path
    - 7 Spring: zero/stub
    - 8 Damper: zero/stub
    - 9 Inertia: zero/stub
    - 10 Friction: zero/stub

    The periodic effects 2, 4, 5, and 6 share one simplified evaluator rather
    than distinct waveform generators.
- linked_workitem: WI-010
- supersedes: none
