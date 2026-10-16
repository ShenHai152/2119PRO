# E-010 — MSFS startup crash and EZFRD in-process isolation

- title: The legacy EZFRD effect driver is loaded in the simulator process
- observed_at: 2026-08-03
- source_type: manual, file, command
- source_ref: User MSFS 2020 startup-crash report; `EZFRD64.dll`; `ffb-bridge/disable-ezfrd-for-games.cmd`
- content_hash: `a36571f371c6c0b1965aa1f0562b3356ddd44bd92566cbd188d17b68af0fa7e8`
- repro_command: |
    ```bash
    export LD_LIBRARY_PATH=/tmp/radare2-local/usr/lib/x86_64-linux-gnu
    /tmp/radare2-local/usr/bin/rabin2 -E /tmp/pxn-unpacked/app/EZFRD64.dll
    /tmp/radare2-local/usr/bin/rabin2 -i /tmp/pxn-unpacked/app/EZFRD64.dll
    /tmp/radare2-local/usr/bin/radare2 -A -q -e scr.color=false \
      -c 'pd 420 @ 0x180001cc0; pd 45 @ 0x1800018a0; pdf @ 0x1800018b0; q' \
      /tmp/pxn-unpacked/app/EZFRD64.dll
    ```
- linked_workitem: WI-009
- supersedes: none

## Evidence

The user reports that MSFS 2020 exits during startup when PXN-F19 is already
attached. X-Plane 12 has not yet been tested for the same behavior.

`EZFRD64.dll` exports `DllRegisterServer` at `0x1800011c0` and
`DllUnregisterServer` at `0x180001480`. Registration creates an
`InProcServer32` COM class and writes `ThreadingModel=Both` for CLSID
`{2D815FAD-11FF-0847-9310-F8608879882B}`. DirectInput therefore loads the
64-bit vendor DLL into a 64-bit game's process when it creates the registered
effect driver. A fault in the DLL appears as a fault of the game executable.

The effect-driver device-initialization method at `0x180001cc0` calls the HID
enumerator at `0x180001600`, opens a raw HID path with `CreateFileA`, obtains
HID preparsed data, and starts a 150 ms multimedia timer with `timeSetEvent`.
The callback address is `0x1800018a0`, which immediately jumps to the effect
scan at `0x180002840`. The destructor at `0x1800018b0` calls `timeKillEvent` for
both stored timer IDs. This timer and raw-HID lifetime is a credible crash
surface inside a simulator process, but the exact MSFS exception remains
unconfirmed until a Windows crash dump or the isolation test is obtained.

The reversible isolation test unregisters only this COM effect class in both
registry views. The joystick's HID input driver remains installed, while the
bridge's native backend enumerates ordinary attached DirectInput devices
without `DIEDFL_FORCEFEEDBACK`, retrieves `DIPROP_GUIDANDPATH`, releases
DirectInput, and writes through the exact HID path.

## Validation boundary

Static analysis validates the in-process loading and risky timer/HID lifetime.
It does not yet prove that this is the exact MSFS exception. Validation requires
running `disable-ezfrd-for-games.cmd`, starting the bridge in service mode, and
launching MSFS 2020 with PXN-F19 already attached. `register-ezfrd.cmd` restores
the original COM registration if input enumeration or native path discovery
does not behave as expected.
