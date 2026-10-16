# E-011 — Encrypted UI resources expose the onboard feature set

- title: Encrypted UI resources expose the onboard feature set
- observed_at: 2026-08-03T15:05:00+08:00
- source_type: file
- source_ref: `/tmp/pxn-unpacked/app/PXNJoystick.exe`, `/tmp/pxn-unpacked/app/uires.zip`
- content_hash: `PXNJoystick.exe=d759147b653e9fe917d811b664acaaff9286e9d78148e9bf8b60eb048817642a`; `uires.zip=f4e86c90266bc07041708adc50635266cdb6f13b00ffd0505d0ae470b96ff8f3`
- repro_command: |
    `rabin2 -zz /tmp/pxn-unpacked/app/PXNJoystick.exe | rg 'lsd123|uires.zip'`
    `bsdtar --passphrase lsd123 -xf /tmp/pxn-unpacked/app/uires.zip -C <temporary-directory>`
    `rg -n 'slider_|btn_fireKey_|btn_Keymap_|btn_Aixymap_|btn_mode' <temporary-directory>/xml/*2119*.xml`
- raw_excerpt: |
    `PXNJoystick.exe` contains the adjacent strings `lsd123` and `uires.zip`.
    The password decrypts the archive successfully. The 2119 layouts expose:
    seven dead-zone sliders, two motor-strength sliders, three mode selectors,
    26 logical mapping slots (21/22 absent from the UI), per-button fire-key
    toggles, axis mapping controls, two POV indicators, RX/RY/Slider axes, and
    save/load controls. `SKeyboardDlg.xml` contains 103 keyboard key IDs;
    `SMouseDlg.xml` contains mouse buttons, movement directions, and wheel input.
- linked_workitem: WI-010
- supersedes: none

The hard-coded archive password is an internal resource-packaging detail, not a
credential belonging to the user. No device I/O was performed while extracting
or reading the resources.
