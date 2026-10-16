# E-004 — EZFRD COM registration and 0x80040154

- Asset: `EZFRD64.dll`
- Functions: `DllGetClassObject` at `0x180001070`; `DllRegisterServer` at `0x1800011c0`
- Tool: radare2 static disassembly

## Evidence

- `DllGetClassObject` compares the requested CLSID with bytes at `0x1801f2b20`:
  `{2D815FAD-11FF-0847-9310-F8608879882B}`.
- `0x180001254`–`0x180001346` constructs `CLSID\{...}\InProcServer32` under `HKEY_CLASSES_ROOT`.
- `0x180001350`–`0x18000138b` writes the module path as the default value.
- `0x180001409`–`0x18000143e` writes `ThreadingModel=Both`.
- `0x18000144f` returns `S_OK` regardless of earlier registry API failures.
- DirectInput `CreateEffect` returning `0x80040154` is `REGDB_E_CLASSNOTREG`: the required in-process class is absent from, or invalid in, the caller's registry view.

## Finding

`regsvr32` must be run elevated for both 64-bit and 32-bit DLLs, and success must
be proven by querying both registry views and checking that each default value
points to the exact matching DLL. The revised registration script performs this
path validation.

