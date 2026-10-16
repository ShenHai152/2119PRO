# E-002 — EZFRD motor output packet

- Asset: `EZFRD64.dll`
- SHA-256: `a36571f371c6c0b1965aa1f0562b3356ddd44bd92566cbd188d17b68af0fa7e8`
- Function: `fcn.180002a60`
- Tool: radare2 static disassembly

## Evidence

- `0x180002b5e`: stores word `0x0500`, producing bytes `00 05`.
- `0x180002b65`: stores byte `01` at packet offset 2.
- `0x180002b91` and `0x180002b95`: store the selected motor amplitude at offsets 3 and 4.
- `0x180002bb2`: sets the output length to 8.
- `0x180002bc4`: calls `WriteFile`.
- Constants at `0x1801b8968` and `0x1801b8970` are `255.0f` and `100.0`, used to scale DirectInput force plus the configured amplifier percentage.

The emitted report is:

```text
00 05 01 AMP AMP 00 00 00
```

## Finding

This is a motor-output report, but it is safe only on the HID handle selected by
EZFRD's DirectInput-instance matching path. The driver chooses the maximum of its
two internally scaled amplitudes and duplicates it into both motor bytes.

