# PXNF/1 simulator vibration service protocol

The PXN bridge defaults to a localhost-only motor service:

```cmd
PXN2119FFBBridge.exe --mode service
```

It listens on UDP `127.0.0.1:21190`. Simulator plugins and adapters own all
effect timing and content. The service only validates commands, writes the
motor, acknowledges packets, stops on timeout, and sends a zero report during
shutdown.

## Packet layout

Every command and acknowledgement is exactly 16 bytes and little-endian.

| Offset | Size | Field | Value |
|---:|---:|---|---|
| 0 | 4 | magic | ASCII `PXNF` |
| 4 | 1 | version | `1` |
| 5 | 1 | command | `1=SET`, `2=STOP`, `3=PING`; ACK sets bit `0x80` |
| 6 | 1 | stick amplitude (`left` in the legacy field name) | `0..255` |
| 7 | 1 | throttle amplitude (`right` in the legacy field name) | `0..255` |
| 8 | 2 | watchdog duration | milliseconds; `0=250`; maximum `5000` |
| 10 | 2 | adapter/state flags | low nibble identifies adapter; high bits identify paused/stopped/exiting |
| 12 | 4 | sequence | monotonically increasing sender sequence |

PXN-F19 uses the two amplitude bytes independently: byte 3 drives the joystick
base/stick motor and byte 4 drives the throttle motor. The native backend writes
them as `00 05 01 STICK THROTTLE 00 00 00`. The binary PXNF/1 layout and legacy
source-field names `left/right` remain unchanged for compatibility.

## Runtime contract

- Send `SET` continuously while an effect is active. A 20–50 Hz update rate is
  recommended.
- The service stops the motor when the packet duration expires. This protects
  against simulator/plugin crashes.
- Send `STOP` when the simulator pauses, unloads the aircraft, disables effects,
  or shuts down.
- Adapters send `PING` at least every 500 ms while idle. The service marks the
  adapter disconnected after 1500 ms without a SET, STOP, or PING.
- A SET acknowledgement confirms that the service accepted the UDP packet. It
  is not a device-health response; commands received while the joystick is
  disconnected are never replayed after reconnection.
- The service only accepts a loopback listen address.
- Forced Task Manager termination cannot execute cleanup; the packet watchdog
  still stops the last command after at most five seconds.

## Adapter and simulator state flags

The low nibble of `flags` identifies the active adapter:

| Value | Adapter |
|---:|---|
| `0` | legacy or unknown PXNF/1 client |
| `1` | MSFS 2020 |
| `2` | MSFS 2024 |
| `3` | X-Plane 12 |
| `4` | generic Microsoft Flight Simulator |

State bits may be combined with the adapter value:

| Bit | Meaning |
|---:|---|
| `0x0100` | simulator paused |
| `0x0200` | simulation stopped or adapter disabled |
| `0x0400` | simulator/adapter exiting |

Old clients that send zero flags remain valid and are displayed as
`Legacy/unknown PXNF/1`.

## Simulator ownership

The protocol deliberately has no definitions for touchdown, stall, runway,
engine, turbulence, or weapon effects. Those belong to the MSFS or X-Plane
adapter/plugin. The motor service receives only the final amplitude and command
duration.
