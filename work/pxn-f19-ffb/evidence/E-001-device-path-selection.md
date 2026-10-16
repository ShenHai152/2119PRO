# E-001 — EZFRD HID path selection

- Asset: `EZFRD64.dll`
- SHA-256: `a36571f371c6c0b1965aa1f0562b3356ddd44bd92566cbd188d17b68af0fa7e8`
- Function: `fcn.180001600`, called by `CMyDIEffectDriver::DeviceID` at `0x180001d28`
- Tool: radare2 static disassembly

## Evidence

- `0x180001659`: calls `DirectInputCreateEx` with version `0x700`.
- `0x1800016ad`: calls `IDirectInputJoyConfig::GetConfig(dwID, ...)` to obtain the configured DirectInput device instance.
- `0x1800016da`–`0x180001737`: obtains the HID class GUID and enumerates every present HID interface with SetupAPI.
- `0x1800017c7`–`0x1800017dd`: copies the candidate SetupAPI device path to the output buffer.
- `0x1800017e9`–`0x1800017f6`: calls DirectInput `FindDevice(..., candidatePath, &candidateInstanceGuid)`.
- `0x180001802`–`0x18000181e`: compares the candidate instance GUID with the configured DirectInput instance GUID.
- Only the matching path survives; non-matching paths are discarded and enumeration continues.

## Finding

EZFRD does not select `VID_11FF/PID_0847/MI_00` by string convention. It binds the
HID path whose DirectInput instance GUID equals the joystick instance selected by
DirectInput configuration. A raw VID/PID/interface guess can therefore open a valid
PXN HID endpoint that is not the force-feedback output endpoint.

