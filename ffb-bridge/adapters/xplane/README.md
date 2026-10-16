# X-Plane 12 adapter

This XPLM plugin registers semantic stick/throttle DataRefs, legacy aliases,
and two control DataRefs:

| DataRef | Type | Meaning |
|---|---|---|
| `pxn/ffb/stick` | float | final joystick-base motor amplitude, `0..1` |
| `pxn/ffb/throttle` | float | final throttle motor amplitude, `0..1` |
| `pxn/ffb/left` | float | legacy alias of `pxn/ffb/stick` |
| `pxn/ffb/right` | float | legacy alias of `pxn/ffb/throttle` |
| `pxn/ffb/enabled` | int | nonzero sends commands; zero sends STOP |
| `pxn/ffb/duration_ms` | int | service watchdog duration, `1..5000` |

Aircraft plugins or scripts own all vibration content and write these DataRefs.
The adapter only forwards final values to the local PXN service at 50 Hz.

The plugin monitors `sim/time/paused`. Pausing, disabling, or unloading the
plugin sends STOP immediately; while idle or paused it sends a 500 ms heartbeat
tagged as X-Plane 12 so the service can display connection state.

Build on Windows with the X-Plane SDK:

```cmd
cmake -S . -B build -DXPLANE_SDK="D:\XPlaneSDK"
cmake --build build --config Release
```

Install the resulting `PXNXPlaneAdapter.xpl` under:

```text
X-Plane 12/Resources/plugins/PXNXPlaneAdapter/64/win.xpl
```
