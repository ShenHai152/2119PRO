# E-003 — DirectInput effect update semantics

- Assets: `EZFRD64.dll`, MinGW `dinput.h`/`dinputd.h`
- Function: `CMyDIEffectDriver::DownloadEffect` at `0x180001e60`; update helper `fcn.180002600`
- Tool: radare2 static disassembly and DirectInput header comparison

## Evidence

- Driver vtable `0x1801b8818` maps offset `0x48` to `DownloadEffect` (`0x180001e60`).
- `0x18000201c` calls `fcn.180002600` for an existing effect.
- `0x18000263a`–`0x1800027bb` tests update mask bits in DirectInput order:
  duration `0x01`, sample period `0x02`, gain `0x04`, trigger `0x08`, repeat `0x10`, axes `0x20`, direction `0x40`, envelope `0x80`, type-specific data bit 8 (`0x100`).
- `0x1800020a7`–`0x1800020c6` tests bit 29 (`0x20000000`) and calls `StartEffect` when `DIEP_START` is present.
- Official header values: `DIEP_TYPESPECIFICPARAMS=0x00000100`, `DIEP_START=0x20000000`.
- Official GUIDs end in `9A-D0-00-A0-C9-A0-6E-35`; the prior bridge used an incorrect tail for both Sine and ConstantForce.

## Finding

The prior bridge sent `0x80|0x20`, which means envelope plus axes, not
type-specific amplitude plus start. Dynamic amplitude changes were therefore not
downloaded to EZFRD. The full effect GUIDs were also malformed. Both defects are
corrected in the rebuilt bridge.

