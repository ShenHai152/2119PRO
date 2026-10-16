# Findings

## F-001 — Direct HID fallback selected the wrong endpoint

Evidence: [E-001](evidence/E-001-device-path-selection.md), [E-002](evidence/E-002-motor-output.md)

The alarm-like sound does not disprove the recovered motor packet. It shows that
writing the packet to a guessed `MI_00` path bypassed EZFRD's required instance
matching and initialization. Native HID fallback remains disabled.

## F-002 — The observed CreateEffect error is a registration failure

Evidence: [E-004](evidence/E-004-com-registration.md)

`0x80040154` is resolved by registering `EZFRD64.dll`/`EZFRD.dll` in their
respective registry views and verifying the registered paths.

## F-003 — The previous bridge could not update vibration amplitude correctly

Evidence: [E-003](evidence/E-003-directinput-effect-update.md)

The prior build used malformed effect GUIDs and wrong `DIEP_*` masks. The new
binary uses the official Sine/ConstantForce GUIDs and sends type-specific updates
with `0x100`.

## F-004 — ConstantForce is the correct deterministic motor test

Evidence: [E-005](evidence/E-005-effect-sampling.md)

EZFRD evaluates effects on a 150 ms multimedia timer. ConstantForce effect ID 0
maps magnitude directly to motor amplitude, while Sine effect ID 3 depends on
timer phase and period. The revised bridge therefore tests ConstantForce first.

## F-005 — DirectInput can select a safe native-output path

Evidence: [E-001](evidence/E-001-device-path-selection.md), [E-002](evidence/E-002-motor-output.md), [E-006](evidence/E-006-directinput-selected-hid-bypass.md), [E-007](evidence/E-007-hardware-vibration-validation.md)

The bridge can bypass the non-responsive EZFRD timer without guessing a HID
collection. `DIPROP_GUIDANDPATH` binds the PXN DirectInput instance to its exact
native path; VID/PID validation and release-before-open preserve the vendor's
selection invariant. The user confirmed physical vibration on the real device;
the revised build lengthens the test pulse and exposes a simulator gain option.

## F-006 — Program exit must explicitly stop the latched motor output

Evidence: [E-008](evidence/E-008-motor-stop-on-process-exit.md)

The PXN-F19 retains the last native motor amplitude after the process handle is
closed. Default Windows process termination skipped Go defers, so the previous
build could exit without sending a zero report. The bridge now handles console
control events and blocks Windows termination until the stop report is sent.

## F-007 — Realistic effect ownership belongs in simulator adapters

Evidence: [E-009](evidence/E-009-simulator-owned-service-interface.md)

MSFS and X-Plane do not provide one common native haptic stream for arbitrary
USB devices. The maintainable boundary is a simulator-owned adapter that emits
final amplitudes to a device-only service. PXNF/1 now supplies that boundary,
including acknowledgements, command lifetime, crash timeout, and shutdown stop.

## F-008 — Isolating EZFRD prevents the reported MSFS startup crash

Evidence: [E-010](evidence/E-010-msfs-ezfrd-inprocess-crash-isolation.md), [E-014](evidence/E-014-msfs-isolation-hardware-validation.md)

`EZFRD64.dll` is an `InProcServer32` DirectInput effect driver. MSFS can load it
inside `FlightSimulator.exe` while enumerating the attached PXN-F19. Its device
initialization opens raw HID state and creates a recurring multimedia timer, so
a vendor-driver lifetime fault is surfaced as a game crash. The reversible
unregister-and-launch test eliminated the reported crash on the user's machine.
The exact exception instruction remains unknown because no crash dump was
captured.

## F-009 — The 2119 Pro has a larger onboard configuration surface than the visible documentation suggests

Evidence: [E-011](evidence/E-011-encrypted-ui-and-onboard-features.md), [E-012](evidence/E-012-onboard-feature-report-protocol.md)

The encrypted UI package and PXN2119 handlers confirm three onboard modes,
per-button rapid-fire state, 24 active remapping slots out of a 26-slot table,
keyboard/mouse/joystick remapping, ten axis-direction mapping slots, seven
independent dead zones, and profile import/export. These settings are uploaded
to the device as 64-byte HID feature reports rather than remaining software-only.

## F-010 — F1/F2/F3/F4 are configuration services, not runtime vibration commands

Evidence: [E-012](evidence/E-012-onboard-feature-report-protocol.md)

F1 uploads logical button mappings, F2 uploads each button's fire-key value, F3
uploads axis mappings, and F4 uploads 0..100 axis parameters for all three modes.
Unknown/reserved parameter IDs must not be probed on real hardware without a
backup-and-restore plan.

## F-011 — EZFRD implements only the non-condition DirectInput effects

Evidence: [E-005](evidence/E-005-effect-sampling.md), [E-013](evidence/E-013-ezfrd-effect-matrix.md)

ConstantForce, RampForce, Square, Sine, Triangle, SawtoothUp, and SawtoothDown
reach executable evaluator paths. Spring, Damper, Inertia, and Friction return
zero. Square/Triangle/Sawtooth share one simplified periodic implementation, so
the original driver cannot deliver distinct high-fidelity condition forces.

## F-012 — The native motor packet supports independent calibrated channels

Evidence: [E-015](evidence/E-015-independent-motor-calibration-and-reconnect.md)

The original UI's motor-preview worker accepts separate joystick-base and
throttle strengths in `00 05 01 STICK THROTTLE 00 00 00`. PXNF/1 already
carried both bytes under the legacy source names `left/right`,
but the bridge previously collapsed them to one maximum and duplicated the
result. The revised service preserves both channels, applies per-channel gain
and minimum-output calibration, and reconnects without replaying stale output.
Automated and cross-build validation passed; independent physical
joystick/throttle actuation remains pending on the Windows device.

## F-013 — Simulator and device lifecycle state is observable without changing PXNF/1 packet size

Evidence: [E-016](evidence/E-016-simulator-identity-and-runtime-status.md)

PXNF/1 keeps its 16-byte layout and uses the reserved flags word for adapter
identity plus paused, stopped, and exiting state. The service reports the active
MSFS 2020, MSFS 2024, X-Plane 12, or generic MSFS adapter; calibrated stick and
throttle amplitudes; device online, disconnected, and reconnected transitions;
and watchdog-stopped state. Simulator pause, stop, exit, adapter timeout, or
device loss clears both motor channels. Go-side behavior and Windows cross-build
are validated; official-SDK adapter compilation and simulator runtime acceptance
remain pending.

## F-014 — The MSFS adapter is now a compiled, service-managed executable

Evidence: [E-017](evidence/E-017-compiled-msfs-adapter-and-supervisor.md), [E-018](evidence/E-018-msfs2020-kittyhawk-runtime-events.md)

`PXNMSFSAdapter.exe` is built as a Windows amd64 Go executable and loads the
Microsoft SimConnect runtime dynamically. The bridge detects
`FlightSimulator.exe` and `FlightSimulator2024.exe`, starts the adjacent adapter
once, and supervises its exit/restart. The adapter identifies MSFS 2020/2024
from `SIMCONNECT_RECV_OPEN`; its testable lifecycle state machine maps Pause,
SimStop, and Quit to zero-output STOP commands and maps Quit to process exit.
Offline self-test, unit tests, Windows cross-build, and archive hash comparison
passed. Real MSFS 2020/2024 runtime acceptance remains pending.

## F-015 — MSFS 2020 identifies itself as KittyHawk 11.x over SimConnect

Evidence: [E-018](evidence/E-018-msfs2020-kittyhawk-runtime-events.md)

Real MSFS 2020 runtime output reports `SIMCONNECT_RECV_OPEN` application name
`KittyHawk` and version 11.0 rather than a public product name. The adapter now
maps this identity to `MSFS 2020`. The same run delivered Pause, SimStop,
SimStart, and resume events, validating the real lifecycle subscription path.
End-to-end service status after replacement and MSFS 2024 identity remain to be
validated.

## F-016 — ClientData-only forwarding cannot vibrate without an effect producer

Evidence: [E-018](evidence/E-018-msfs2020-kittyhawk-runtime-events.md), [E-019](evidence/E-019-msfs-simvar-vibration-mixer.md)

MSFS delivered OPEN and lifecycle events, but it does not automatically write
final motor amplitudes into the custom `PXN2119_FFB` ClientData area. Therefore
the earlier forwarding-only adapter remained at stick=0/throttle=0 unless an
aircraft or WASM producer was separately installed. The revised adapter
subscribes directly to MSFS SimVars and generates final independent amplitudes:
airframe/stall/ground effects on the joystick-base motor and engine/powerplant
effects on the throttle motor. External ClientData commands retain temporary
priority. Portable tests and Windows cross-build passed; real-flight amplitude
acceptance remains pending.

## F-017 — Expired Windows UDP deadlines and latched pause can suppress all output

Evidence: [E-020](evidence/E-020-windows-udp-poll-and-simstart-resume-fix.md)

The real adapter received SimConnect events, but the service remained
`adapter=none`, showing that packets were not reaching the decoder. The service
set an already-expired read deadline immediately before every Windows UDP read;
this is replaced with a one-millisecond future deadline. Separately, SimStart
now clears a previously latched Pause state because the observed event sequence
did not always include a later resume event. Explicit service-ACK and telemetry
stream diagnostics were added. Windows runtime confirmation remains pending.

## F-018 — Sustained high dual-motor output needs a final hardware safety envelope

Evidence: [E-021](evidence/E-021-final-motor-safety-envelope.md)

The user observed that MSFS telemetry gain 2.0 produced vibration briefly and
then disconnected PXN-F19, while gain 1.5 remained stable. The exact electrical
cause has not been measured, but the gain threshold and recovery behavior are
consistent with USB voltage drop, controller current protection, or firmware
motor protection. Because saved calibration can amplify either channel and any
adapter can request 255/255, protection belongs after calibration at the final
HID boundary rather than only in the MSFS mixer.

The service now defaults to per-channel maximums of 220, a combined budget of
330, and a rising step of 24 per output update. Shared-budget scaling is
proportional, reductions are immediate, and status displays the bytes actually
written. Automated tests and Windows cross-builds are validated; the new limits
remain candidate hardware protection until a sustained real-flight test passes.

## F-019 — Reverse thrust requires adapter-owned signed axis mapping

Evidence: [E-012](evidence/E-012-onboard-feature-report-protocol.md), [E-022](evidence/E-022-msfs-reverse-zone-throttle.md)

The recovered onboard protocol exposes the physical Z/throttle object, axis
mapping, and dead-zone configuration, but no negative-range or reverse-detent
firmware setting. The MSFS adapter therefore implements reverse as an optional
software curve over the same physical Z axis. The default advanced profile maps
the bottom 12% from full reverse to idle, keeps the next 2% at idle, and rescales
the remaining travel to full forward. It supports either raw-axis direction.

Because the adapter transmits the entire signed `AXIS_THROTTLE_SET` range, the
normal PXN-F19 throttle-axis binding must be removed from MSFS while this mode is
enabled. Unit tests and Windows cross-builds are validated; real-aircraft
behavior is candidate, especially for aircraft that replace generic throttle
events with custom WASM logic.

## F-020 — A negative generic throttle axis can be accepted but clamped at idle

Evidence: [E-023](evidence/E-023-msfs-negative-axis-requires-reverse-event.md)

The real MSFS 2020 log repeatedly reported raw 65535, physical position 0%, and
signed output -16383 in reverse mode. This validates device selection, axis
direction, and curve calculation, but the tested aircraft still did not deploy
reverse thrust. The failure is therefore after input mapping: the aircraft or
simulator requires an explicit reverser-unlock event and/or per-engine axis
events rather than only `AXIS_THROTTLE_SET`.

The adapter now sends generic plus engine 1–4 throttle axes and defaults to a
repeated `THROTTLE_REVERSE_THRUST_HOLD` while inside the reverse zone. A
transition-safe Toggle mode is available for aircraft that latch their
reverser. Automated tests and Windows builds pass; physical reverse deployment
with the revised executable is still pending.

## Path P-009 — MSFS SimVars to independent physical motors

```text
MSFS SimConnect SIM_FRAME telemetry
  -> acceleration / G force / stall / on-ground / ground velocity
  -> engine RPM / turbine N1 / combustion
  -> smoothed and clamped two-channel mixer
  -> PXNF/1 SET(stick, throttle, 150 ms)
  -> service calibration + watchdog
  -> joystick-base motor and throttle motor

Pause / SimStop / Quit
  -> reset mixer
  -> immediate PXNF/1 STOP(0, 0)
```

## Path P-010 — Calibrated simulator request through final motor protection

```text
PXNF/1 SET(stick, throttle)
  -> independent saved gain and minimum-output calibration
  -> per-channel maximums (default 220 / 220)
  -> proportional shared budget (default 330)
  -> rising slew (default 24 per update)
  -> DirectInput-selected native HID write
  -> actual stick/throttle amplitudes shown in Status

STOP / lower request / Pause / SimStop / Quit / watchdog / disconnect
  -> immediate falling edge
  -> 00 05 01 00 00 00 00 00
```

## Path P-011 — Physical PXN throttle travel to signed MSFS reverse thrust

```text
PXN-F19 DirectInput device
  -> background/nonexclusive Z/Throttle object, normalized 0..65535
  -> configurable high/low physical idle direction
  -> bottom 12%: -100% reverse to idle
  -> next 2%: neutral/idle
  -> remaining 86%: idle to 100% forward
  -> signed SimConnect AXIS_THROTTLE_SET (-16383..16383)
  -> MSFS 2020/2024 aircraft throttle

Pause / SimStop / Quit / input loss / adapter shutdown
  -> immediate AXIS_THROTTLE_SET(0)
```

## Path P-008 — MSFS process detection to managed SimConnect adapter lifecycle

```text
FlightSimulator.exe or FlightSimulator2024.exe appears
  -> PXN bridge process supervisor
  -> start adjacent PXNMSFSAdapter.exe once
  -> dynamic SimConnect.dll load and connection retry
  -> SIMCONNECT_RECV_OPEN identifies MSFS 2020 or MSFS 2024
  -> Pause / SimStop / Quit sends PXNF/1 STOP(stick=0, throttle=0)
  -> Quit or simulator-process fallback exits adapter
  -> supervisor does not restart until an MSFS process exists
```

## Path P-007 — Simulator lifecycle to status display and safe motor stop

```text
MSFS OPEN/Pause/SimStart/SimStop/Quit or X-Plane paused/disable/unload
  -> adapter identity + paused/stopped/exiting flags
  -> SET/STOP plus 500 ms idle heartbeat
  -> PXNF/1 service adapter tracker + 1500 ms adapter timeout
  -> device health, adapter, simulator state, stick/throttle status display
  -> STOP clears both joystick-base and throttle motor channels

Device unplug or idle HID health-check failure
  -> disconnected status + zero logical output
  -> reconnect polling
  -> reconnected status + immediate zero report
  -> wait for a fresh simulator SET before resuming output
```

## Path P-006 — Simulator channels through calibration and reconnect

```text
PXNF/1 SET(stick/left, throttle/right, duration, sequence)
  -> packet validation and watchdog state
  -> per-channel gain and minimum-output calibration
  -> reconnecting motor controller
  -> DirectInput-selected HID collection
  -> WriteFile(00 05 01 STICK THROTTLE 00 00 00)
  -> independent PXN-F19 motors

Disconnect:
WriteFile failure or unplug
  -> close handle and discard queued SET commands
  -> retry exact DirectInput path every second
  -> send zero immediately after reconnect
  -> wait for a fresh SET before motor output
```

## Path P-005 — UI setting to onboard profile upload

```text
encrypted uires.zip layouts
  -> PXN2119 UI event handlers
  -> three per-mode 0xF8-byte configuration records
  -> F1 button mapping / F2 fire-key / F3 axis mapping / F4 dead-zone reports
  -> HidD_SetFeature(..., 64)
  -> PXN-F19 onboard mode 0, 1, or 2
```

## Path P-004 — Legacy driver crash isolation

```text
PXN-F19 attached before MSFS startup
  -> DirectInput discovers the registered FFB effect class
  -> COM loads EZFRD64.dll inside FlightSimulator.exe
  -> EZFRD initializes raw HID state and a 150 ms multimedia timer
  -> suspected vendor DLL lifetime fault appears as an MSFS crash

Isolation:
disable-ezfrd-for-games.cmd
  -> unregister 32/64-bit EZFRD COM class
  -> MSFS keeps ordinary HID axes/buttons without loading EZFRD
  -> PXN service finds the ordinary DirectInput instance
  -> DIPROP_GUIDANDPATH selects the exact HID output path
  -> simulator adapter owns vibration through PXNF/1
```

## Path P-003 — Simulator-owned vibration command

```text
MSFS aircraft/WASM content or X-Plane aircraft/plugin content
  -> simulator-specific adapter
  -> PXNF/1 SET(stick/left, throttle/right, duration, sequence) on localhost
  -> PXN motor service validation and watchdog
  -> DirectInput-selected PXN-F19 HID path
  -> WriteFile(00 05 01 STICK THROTTLE 00 00 00)
  -> PXN-F19 motor
```

## Path P-001 — Simulator telemetry to motor report

```text
MSFS 2020/2024 SimConnect or X-Plane 12 RREF
  -> bridge vibration mixer
  -> IDirectInputEffect::SetParameters(DIEP_TYPESPECIFICPARAMS)
  -> DirectInput IDirectInputEffectDriver::DownloadEffect
  -> EZFRD effect state/timer
  -> EZFRD matched HID handle
  -> WriteFile(00 05 01 AMP AMP 00 00 00) through EZFRD
  -> PXN-F19 motors
```

## Path P-002 — Safe timer bypass

```text
PXN-F19 DirectInput instance GUID
  -> IDirectInputDevice8::GetProperty(DIPROP_GUIDANDPATH)
  -> exact HID path validated as VID_11FF/PID_0847
  -> DirectInput objects released
  -> CreateFileW(exact path, read/write + shared)
  -> WriteFile(00 05 01 STICK THROTTLE 00 00 00)
  -> PXN-F19 motors
```
