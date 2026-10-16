# PXN 2119 Pro FFB bridge

`PXNJoystick setupV3.0.exe` contains the vendor's DirectInput Ring-3 effect
driver, `EZFRD.dll`/`EZFRD64.dll`. The bridge supports the normal DirectInput
effect route and an explicit native-output test route.

Do not guess a `VID_11FF/PID_0847/MI_*` path. Real-device testing showed that a
guessed collection can produce an alarm-like sound rather than vibration. The
`hidpath` backend instead asks DirectInput for `DIPROP_GUIDANDPATH`, validates
the returned VID/PID, releases DirectInput, and only then opens that exact path.

## Preventing simulator startup crashes

`EZFRD64.dll` is an old in-process DirectInput effect driver. A game that
enumerates force-feedback devices loads this DLL directly into the game
process. If MSFS 2020 exits while PXN-F19 is attached, close PXNJoystick and run
the following script from an Administrator Command Prompt:

```cmd
disable-ezfrd-for-games.cmd
```

This unregisters only the EZFRD COM effect class. It does not disable the
joystick's normal axes or buttons. Native service mode now enumerates PXN-F19
without the `DIEDFL_FORCEFEEDBACK` filter, so it can still obtain the exact HID
path and drive vibration while the legacy DLL is isolated from MSFS.

Validate the isolation in this order:

```cmd
disable-ezfrd-for-games.cmd
PXN2119FFBBridge.exe --mode service
```

Leave the bridge running, then launch MSFS 2020 with PXN-F19 already attached.
Verify that MSFS reaches the main menu, axes/buttons still respond, and the
bridge reports `PXN-F19 connected through DirectInput-selected native HID`.
This test is also applicable to MSFS 2024. Test X-Plane 12 separately because
its DirectInput initialization behavior may differ.

If the bridge cannot find the PXN-F19 DirectInput instance after isolation,
restore the original registration immediately with `register-ezfrd.cmd` and
report the complete bridge output.

To restore the vendor effect driver later, run:

```cmd
register-ezfrd.cmd
```

## Build on Windows

Install Go 1.21 or newer, then run:

```powershell
$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags="-s -w" -o PXN2119FFBBridge.exe .
go build -trimpath -ldflags="-s -w" -o PXNMSFSAdapter.exe ./cmd/pxn-msfs-adapter
```

Run it from the PXNJoystick installation directory. Service mode is the
default and does not calculate effects from telemetry:

```powershell
.\PXN2119FFBBridge.exe --mode service
```

Expected startup output:

```text
PXN-F19 connected through DirectInput-selected native HID
PXN motor service listening on udp://127.0.0.1:21190 (PXNF/1)
Vibration content is controlled by the simulator adapter
MSFS adapter supervision enabled: D:\Program Files\PXNJoystick\PXNMSFSAdapter.exe
```

Service mode automatically reports simulator and hardware state once per
second:

```text
Simulator adapter identified: MSFS 2024
Status: device=online adapter=MSFS 2024 simulator=active stick=96 throttle=140
Status: device=online adapter=MSFS 2024 simulator=paused stick=0 throttle=0
```

MSFS 2020/2024 are identified from the SimConnect server name/version.
X-Plane identifies itself as X-Plane 12. Pause, SimStop, plugin disable, and
simulator exit send an immediate STOP. A 500 ms adapter heartbeat and 1500 ms
service timeout handle abnormal adapter exits. Use `--status-interval 0` to
disable periodic lines or specify another duration such as `500ms`.

Service mode also monitors `FlightSimulator.exe` and
`FlightSimulator2024.exe`. If `PXNMSFSAdapter.exe` is beside the bridge, it is
started automatically when MSFS appears and exits with SimConnect `Quit` or the
simulator-process fallback. Use `--msfs-adapter off` to disable supervision, or
pass an explicit executable path. The adapter's offline lifecycle check is:

```cmd
PXNMSFSAdapter.exe --self-test
```

`PXNMSFSAdapter.exe` dynamically loads Microsoft's `SimConnect.dll`. If the
runtime is not already discoverable by Windows, copy the SDK file beside the
adapter or set `MSFS_SDK` to the SDK root.

The MSFS adapter now subscribes directly to simulator SimVars. By default the
joystick-base motor represents airframe load, stall, acceleration changes,
touchdown, and ground roll; the throttle motor represents engine RPM/N1 and
powerplant vibration. Tune automatic operation from the bridge command line:

The adapter is vibration-only: it does not read, remap, or write any joystick,
throttle, reverse-thrust, or other flight-control input. Keep the normal
PXN-F19 control bindings configured directly inside the simulator.

```cmd
PXN2119FFBBridge.exe --mode service --msfs-telemetry-gain 1.5
PXN2119FFBBridge.exe --mode service --msfs-telemetry=false
```

The adapter logs its generated values once per second as
`MSFS telemetry: stick=... throttle=...`. Pause, SimStop, and Quit still send
immediate zero-output STOP commands.

Real-device testing found that telemetry gain `2.0` could sustain enough motor
load to disconnect PXN-F19, while `1.5` remained stable. Service mode therefore
applies a final hardware safety envelope after per-channel calibration and
before every native HID write. The defaults cap each motor at 220, cap their
combined output at 330, and limit rising edges to 24 amplitude units per output
update. Falling edges are never ramped, so Pause, SimStop, Quit, watchdog expiry,
and adapter disconnect still stop both motors immediately.

The startup line reports the active envelope:

```text
Motor safety: stick-max=220 throttle-max=220 budget=330 slew=24/update
```

Advanced overrides are available for hardware tuning:

```cmd
PXN2119FFBBridge.exe --mode service --msfs-telemetry-gain 1.5 --stick-max 220 --throttle-max 220 --motor-budget 330 --motor-slew 24
```

Keep `--msfs-telemetry-gain` at or below `1.5` for the first hardware test. If a
request exceeds the envelope, the bridge rate-limits a diagnostic such as
`Motor safety limit: requested ... applied-target ...`; the periodic status line
continues to show the amplitudes actually written to the motors.

The service accepts the documented 16-byte `PXNF/1` protocol. See
[PXNF-PROTOCOL.md](PXNF-PROTOCOL.md) and [pxn_ffb_protocol.h](pxn_ffb_protocol.h).
It applies only the final amplitudes supplied by the simulator adapter. The
recovered native packet supports independent joystick-base and throttle motor
bytes, so PXNF/1 now preserves both channels end to end. The packet retains the
legacy field names `left/right`, mapped as `left=stick` and `right=throttle`.

If PXN-F19 is unplugged or a HID write fails, service mode remains alive and
retries the exact DirectInput-selected path every second. It writes zero after
reconnection and does not replay commands received while disconnected. An HID
health check also detects unplugging while the simulator is idle.

Verify the service interface from a second PowerShell window:

```powershell
.\send-vibration.ps1 -Amplitude 255 -Seconds 3
.\send-vibration.ps1 -Stick 180 -Throttle 0 -Seconds 3
.\send-vibration.ps1 -Stick 0 -Throttle 180 -Seconds 3
```

## Motor calibration

Calibration is stored by default in
`%LOCALAPPDATA%\PXN2119FFBBridge\config.json`. Missing or invalid configuration
falls back to linear gain 1.0 with no minimum-output lift.

Run the interactive stick/throttle start-threshold and gain-balancing wizard:

```cmd
PXN2119FFBBridge.exe --mode calibrate
```

Use this wizard when the joystick-base and throttle motors feel different. They
are separate physical eccentric motors and the simulator intentionally sends
different content to them, so do not expect identical vibration at every moment.
The wizard records an independent `gain` and `minimum_output` for each channel;
the final safety envelope remains active in service mode even when calibration
raises one channel's gain.

Reset to linear defaults:

```cmd
PXN2119FFBBridge.exe --mode calibrate --reset
```

Run a one-shot channel test. Saved calibration is applied unless `--raw` is
specified:

```cmd
PXN2119FFBBridge.exe --mode motor-test --channel stick --amplitude 80 --duration 3s
PXN2119FFBBridge.exe --mode motor-test --channel throttle --amplitude 80 --duration 3s
PXN2119FFBBridge.exe --mode motor-test --channel both --amplitude 160 --duration 3s
```

The former `left/right` channel names remain accepted as aliases.

Use `--config D:\path\config.json` to override the default calibration file.

MSFS and X-Plane do not expose a universal native haptic-output stream. To
make simulator or aircraft content own the effects, use a simulator-side
adapter:

- [MSFS 2020/2024 SimConnect ClientData adapter](adapters/msfs/README.md)
- [X-Plane 12 writable-DataRef adapter](adapters/xplane/README.md)

The ready-to-run MSFS adapter is included in the release package. Its Go build
does not need SDK headers or `SimConnect.lib`, although Microsoft's
`SimConnect.dll` is still required at runtime. It can generate amplitudes from
MSFS SimVars or accept final ClientData commands from aircraft content. The
X-Plane plugin source still requires the X-Plane SDK to build.

If DirectInput reports `ConstantForce` but the motor remains still, test the
DirectInput-selected native path:

```cmd
PXN2119FFBBridge.exe --mode test --output hidpath
```

The expected result is `PXN-F19 connected through DirectInput-selected native
HID`, followed by a 1.5-second vibration every 3 seconds. This mode sends the
verified report `00 05 01 STICK THROTTLE 00 00 00` and uses the full amplitude byte
range `0..255`. Real-device validation succeeded, so `hidpath` is now the
default. Use `--output directinput` only for EZFRD diagnostics.

The test pulse is already full amplitude. To give the motors more spin-up time,
run a five-second pulse every six seconds:

```cmd
PXN2119FFBBridge.exe --mode test --output hidpath --test-on 5s --test-cycle 6s
```

The previous built-in telemetry mixer remains available only as a legacy
compatibility path:

```cmd
PXN2119FFBBridge.exe --mode msfs --gain 3
PXN2119FFBBridge.exe --mode xplane --gain 3
```

These legacy modes are not used by service mode and are not the recommended
architecture when simulator content should own the vibration.

Stop the bridge with Ctrl+C or close its console window. The bridge handles
Windows Ctrl+C, Ctrl+Break, console-close, logoff, and shutdown events and sends
`00 05 01 00 00 00 00 00` before releasing the process. Avoid terminating it
from Task Manager; forced process termination cannot run motor-stop cleanup.

Completely exit `PXNJoystick.exe` before starting the bridge so it does not
compete with the bridge for the DirectInput effect device.

If effect creation reports `0x80040154`, open an Administrator Command Prompt
in the PXNJoystick installation directory and run `register-ezfrd.cmd`.  The
vendor DLL can return success even when a non-elevated registry write failed,
so the script verifies that both registry views point to the exact DLLs beside
the script for CLSID `{2D815FAD-11FF-0847-9310-F8608879882B}`. Then unplug and
reconnect the joystick and run:

```cmd
PXN2119FFBBridge.exe --mode test
```

Use `--output directinput` only for EZFRD diagnostics. Normal service and test
operation use the real-device-validated `hidpath` backend.
