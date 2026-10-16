# MSFS 2020/2024 adapter

The release package now includes a ready-to-run `PXNMSFSAdapter.exe`. It is
built in Go and loads `SimConnect.dll` dynamically, so compiling the adapter no
longer requires Visual Studio, the MSFS SDK headers, or `SimConnect.lib`.

This adapter actively connects to SimConnect and, by default, subscribes to
MSFS SimVars to generate final two-channel `PXNF/1` amplitudes. The joystick
base receives airframe load, acceleration changes, stall warning, touchdown,
and ground-roll vibration. The throttle receives engine RPM/N1 and powerplant
vibration. Pause, SimStop, and Quit always override telemetry with STOP.

This adapter is vibration-only. It never writes joystick axes, throttle axes,
reverse-thrust events, or any other aircraft control. All PXN-F19 flight-control
bindings remain owned by MSFS.

Aircraft packages and WASM modules can still write final 16-byte commands to
the SimConnect ClientData area named `PXN2119_FFB`. A received SET/STOP command
temporarily takes priority over the built-in SimVar mixer for its duration.

For PXN-F19, the packet's legacy `left` field drives the joystick-base motor
and `right` drives the throttle motor. The binary PXNF/1 layout is unchanged.

## Automatic start and exit

Keep `PXNMSFSAdapter.exe` beside `PXN2119FFBBridge.exe` and start service mode:

```cmd
PXN2119FFBBridge.exe --mode service
```

The bridge monitors `FlightSimulator.exe` and `FlightSimulator2024.exe`. It
starts one adapter process when either simulator appears. The adapter retries
SimConnect while the simulator starts, identifies the exact version from
`SIMCONNECT_RECV_OPEN`, and exits on SimConnect `Quit`. A process-liveness
fallback also exits the adapter if the simulator crashes without delivering
`Quit`.

Disable automatic launch or specify another adapter path with:

```cmd
PXN2119FFBBridge.exe --mode service --msfs-adapter off
PXN2119FFBBridge.exe --mode service --msfs-adapter "D:\Tools\PXNMSFSAdapter.exe"
```

Manual adapter launch remains available:

```cmd
PXNMSFSAdapter.exe --service 127.0.0.1:21190
PXNMSFSAdapter.exe --service 127.0.0.1:21190 --telemetry-gain 1.5
```

Use `--telemetry=false` only when an aircraft/WASM package supplies all final
PXNF/1 commands itself. The default telemetry update interval is 50 ms and can
be changed with `--telemetry-interval`.

## Validate lifecycle handling

Run the offline state-machine test without starting MSFS:

```cmd
PXNMSFSAdapter.exe --self-test
```

Expected results include:

```text
Self-test PASS: MSFS 2020/2024 detection
Self-test PASS: Pause -> STOP(stick=0, throttle=0)
Self-test PASS: SimStop -> STOP(stick=0, throttle=0)
Self-test PASS: Quit -> STOP(stick=0, throttle=0) + exit
```

In the simulator, the adapter logs `MSFS Pause`, `MSFS SimStop`, and
`MSFS Quit`; all three paths send zero for both motor channels before changing
state or exiting.

## SimConnect runtime

The adapter still requires Microsoft's `SimConnect.dll` at runtime, as every
external SimConnect client does. It first checks beside `PXNMSFSAdapter.exe`,
then the normal Windows DLL search path, then an SDK path supplied through the
`MSFS_SDK` environment variable. If Windows cannot locate the DLL, copy the
SDK's `SimConnect SDK\lib\SimConnect.dll` beside the adapter.

## Build the shipped Go adapter

From the bridge repository root with Go 1.21 or newer:

```powershell
$env:GOOS = 'windows'
$env:GOARCH = 'amd64'
$env:CGO_ENABLED = '0'
go build -trimpath -ldflags='-s -w' -o PXNMSFSAdapter.exe ./cmd/pxn-msfs-adapter
```

The older SDK-linked C++ implementation remains as an ABI reference. It can
still be built from a Visual Studio developer prompt if required:

```cmd
cmake -S . -B build -DMSFS_SDK="D:\MSFS SDK"
cmake --build build --config Release
```

The same ClientData contract applies to MSFS 2020 and MSFS 2024.

The adapter identifies MSFS 2020 or 2024 from the SimConnect OPEN message and
adds that identity to every PXNF/1 packet. It subscribes to `Pause`, `SimStart`,
and `SimStop`; pause, SimStop, and simulator quit send STOP immediately. When no
effect packet is flowing, a 500 ms PING heartbeat keeps the service-side
connection state current.

MSFS 2020 reports the internal application name `KittyHawk` with application
version 11.x; this is explicitly mapped to `MSFS 2020`.
