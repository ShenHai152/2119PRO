# E-018 — MSFS 2020 KittyHawk runtime identity and lifecycle events

- title: Real MSFS 2020 SimConnect OPEN identity and lifecycle events
- observed_at: 2026-08-03T19:35:00+08:00
- source_type: log
- source_ref: user Windows console running `PXNMSFSAdapter.exe --service 127.0.0.1:21190`
- content_hash: `PXNMSFSAdapter.exe=d39aa9421f60bf6f6ecd34649b9a0cc96c42e3fb081356ad601b5222d49e9b6a`; `PXN2119-FFB-Bridge.zip=8a2edb3c7048c3fe6a90fc081e34f5e110c260e94d0a197a6614090fdf77d74c`
- repro_command: |
    `PXNMSFSAdapter.exe --service 127.0.0.1:21190`
- raw_excerpt: |
    SimConnect runtime loaded: D:\Program Files\PXNJoystick\SimConnect.dll
    SimConnect connected: KittyHawk version 11.0; detected Microsoft Flight Simulator
    MSFS Pause: stopping stick and throttle motors
    MSFS SimStop: stopping stick and throttle motors
    MSFS SimStart
    MSFS resumed
    MSFS Pause: stopping stick and throttle motors
    MSFS resumed
- linked_workitem: WI-013
- supersedes: E-017

The runtime log validates real SimConnect OPEN, Pause, SimStop, SimStart, and
resume dispatch on MSFS 2020. It also shows that MSFS 2020 exposes the internal
application name `KittyHawk` with version 11.0. The adapter detection table now
maps `KittyHawk` explicitly to MSFS 2020. End-to-end bridge status and automatic
child launch still require acceptance after replacing the adapter executable.
