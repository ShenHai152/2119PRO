# E-005 — Effect sampling and deterministic test selection

- Asset: `EZFRD64.dll`
- Functions: `CMyDIEffectDriver::StartEffect` at `0x180002210`, timer callback at `0x180002840`, force evaluator at `0x180003210`
- Tool: radare2 static disassembly

## Evidence

- `0x18000230c`: `StartEffect` changes the selected effect slot state to running (`1`).
- `0x180002317`: records the start tick.
- `0x180002840` is reached from the multimedia timer callback created by `DeviceID` with a 150 ms delay.
- `0x180002920`–`0x1800029f4`: the timer scans running effect slots, calculates elapsed time and evaluates each effect.
- `0x180003250`–`0x180003277`: dispatches by effect ID.
- Switch table `0x1800034e4` maps effect ID 0 to `0x180003279`, which reads the first type-specific DWORD (ConstantForce magnitude) and scales it directly.
- Effect ID 3 maps to `0x1800033bf`, which calculates a sine sample from phase, period and elapsed time.

## Finding

ConstantForce is the most deterministic hardware-test effect because it maps
directly to motor amplitude. A short Sine test can be distorted or aliased by
EZFRD's 150 ms evaluation cadence. The bridge now tries ConstantForce first and
uses Sine only as a fallback.

