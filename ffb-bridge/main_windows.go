//go:build windows

// PXN 2119 Pro force-feedback service. Simulator adapters own vibration
// content and send final motor commands through the localhost PXNF/1 API.
// Legacy MSFS/SimConnect and X-Plane/RREF telemetry modes remain available.
package main

import (
	"encoding/binary"
	"errors"
	"flag"
	"fmt"
	"math"
	"net"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

type guid struct {
	Data1        uint32
	Data2, Data3 uint16
	Data4        [8]byte
}

var iidDirectInput8 = guid{0xBF798031, 0x483A, 0x4DA2, [8]byte{0xAA, 0x99, 0x5D, 0x64, 0xED, 0x36, 0x97, 0x00}}
var guidXAxis = guid{0xA36D02E0, 0xC9F3, 0x11CF, [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}
var guidYAxis = guid{0xA36D02E1, 0xC9F3, 0x11CF, [8]byte{0xBF, 0xC7, 0x44, 0x45, 0x53, 0x54, 0x00, 0x00}}

// DirectInput effect GUIDs.  The original values used here were shifted:
// 0x13541C20 is ConstantForce, while Sine is 0x13541C23.
var guidSine = guid{0x13541C23, 0x8E33, 0x11D0, [8]byte{0x9A, 0xD0, 0x00, 0xA0, 0xC9, 0xA0, 0x6E, 0x35}}
var guidConstant = guid{0x13541C20, 0x8E33, 0x11D0, [8]byte{0x9A, 0xD0, 0x00, 0xA0, 0xC9, 0xA0, 0x6E, 0x35}}

const (
	diEnumDevice       = 4
	diCreateDevice     = 3
	diGetProperty      = 5
	diCreateEffect     = 18
	diSetCooperative   = 13
	diSetDataFormat    = 11
	diAcquire          = 7
	diSetEffectParams  = 6
	diStartEffect      = 7
	diRelease          = 2
	diEnumContinue     = 1
	dieffCartesian     = 0x10
	dieffObjectOffsets = 0x02
	// This is a DIEP_* parameter-update flag, not a DIEFF_* coordinate flag.
	// EZFRD's DownloadEffect implementation tests bit 8 for the type-specific
	// payload.
	diepTypespecific    = 0x00000100
	diFfNominalMax      = 10000
	diInfinite          = 0xffffffff
	diJoofsX            = 0
	diJoofsY            = 4
	diDevClassAll       = 0 // DI8DEVCLASS_ALL
	diDevTypeJoystick   = 4
	diDevTypeGamepad    = 5
	diedflAttachedOnly  = 0x00000001
	diedflForceFeedback = 0x00000100
	disclExclusive      = 0x00000001
	disclBackground     = 0x00000008
	didfAbsAxis         = 0x00000001
	didftAbsAxis        = 0x00000002
	didftAnyInstance    = 0x00ffff00
	dipropGuidAndPath   = 12
	diphDevice          = 0
)

type diDeviceInstance struct {
	Size         uint32
	Guid         guid
	Product      guid
	DevType      uint32
	InstanceName [260]uint16
	ProductName  [260]uint16
	FFDriver     guid
	UsagePage    uint16
	Usage        uint16
}
type diEffect struct {
	Size, Flags, Duration, Sample, Gain, TriggerBtn, TriggerRep, Axes uint32
	RGDAxes, Directions, Envelope                                     uintptr
	TypeSize                                                          uint32
	TypeParams                                                        uintptr
	StartDelay                                                        uint32
}
type diPeriodic struct {
	Magnitude     uint32
	Offset        int32
	Phase, Period uint32
}
type diConstant struct{ Magnitude int32 }
type diObjectDataFormat struct {
	Guid   uintptr
	Offset uint32
	Type   uint32
	Flags  uint32
}
type diDataFormat struct {
	Size       uint32
	ObjectSize uint32
	Flags      uint32
	DataSize   uint32
	Objects    uint32
	ObjectData uintptr
}
type diPropHeader struct {
	Size       uint32
	HeaderSize uint32
	Obj        uint32
	How        uint32
}
type diPropGuidAndPath struct {
	Header diPropHeader
	Class  guid
	Path   [260]uint16
}
type telemetry struct{ GX, GY, GZ, Airspeed, Vertical, Ground, Stall float64 }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func comCall(obj uintptr, index int, args ...uintptr) uintptr {
	vt := *(*uintptr)(unsafe.Pointer(obj))
	table := (*[64]uintptr)(unsafe.Pointer(vt))
	all := append([]uintptr{obj}, args...)
	r, _, _ := syscall.SyscallN(table[index], all...)
	return r
}

type ffb struct {
	dinput, device, effect uintptr
	axes                   [2]uint32
	directions             [2]int32
	periodic               diPeriodic
	constant               diConstant
	usingPeriodic          bool
	lastSetHR              uint32
}

// PXN-F19 / PXN-2119 Pro native USB identity and motor protocol recovered
// from PXNJoystick.exe's CDeviceHandler2119 implementation.
const (
	pxnVID              = "vid_11ff"
	pxnPID              = "pid_0847"
	genericRead         = 0x80000000
	genericWrite        = 0x40000000
	fileShareRead       = 0x00000001
	fileShareWrite      = 0x00000002
	openExisting        = 3
	fileAttributeNormal = 0x00000080
)

type hidMotor struct {
	handle syscall.Handle
}

type hidAttributes struct {
	Size          uint32
	VendorID      uint16
	ProductID     uint16
	VersionNumber uint16
}

func newHIDMotor() (*hidMotor, error) {
	// Match the vendor driver's selection logic: ask DirectInput for the HID
	// path belonging to the exact DirectInput instance.  This avoids guessing
	// an MI_* collection, which previously opened an alarm/buzzer endpoint.
	dll := syscall.NewLazyDLL("dinput8.dll")
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("load dinput8.dll: %w", err)
	}
	proc := dll.NewProc("DirectInput8Create")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	module, _, _ := kernel32.NewProc("GetModuleHandleW").Call(0)
	var di uintptr
	hr, _, _ := proc.Call(module, 0x0800, uintptr(unsafe.Pointer(&iidDirectInput8)), uintptr(unsafe.Pointer(&di)), 0)
	if hr != 0 || di == 0 {
		return nil, fmt.Errorf("DirectInput8Create failed: 0x%x", hr)
	}

	selectedDevice = nil
	fallbackDevice = nil
	cb := syscall.NewCallback(enumCallback)
	// Native service mode only needs the ordinary DirectInput device identity.
	// Do not request DIEDFL_FORCEFEEDBACK here: that flag ties path discovery to
	// the legacy EZFRD effect driver and prevents safe game-crash isolation by
	// unregistering the in-process COM DLL.
	hr = comCall(di, diEnumDevice, diDevClassAll, cb, 0, diedflAttachedOnly)
	if hr != 0 || selectedDevice == nil {
		comCall(di, diRelease)
		return nil, fmt.Errorf("PXN-F19 DirectInput instance not found (EnumDevices HRESULT 0x%x)", hr)
	}
	var dev uintptr
	hr = comCall(di, diCreateDevice, uintptr(unsafe.Pointer(&selectedDevice.Guid)), uintptr(unsafe.Pointer(&dev)), 0)
	if hr != 0 || dev == 0 {
		comCall(di, diRelease)
		return nil, fmt.Errorf("CreateDevice failed: 0x%x", hr)
	}

	property := diPropGuidAndPath{}
	property.Header.Size = uint32(unsafe.Sizeof(property))
	property.Header.HeaderSize = uint32(unsafe.Sizeof(property.Header))
	property.Header.How = diphDevice
	hr = comCall(dev, diGetProperty, uintptr(dipropGuidAndPath), uintptr(unsafe.Pointer(&property)))
	if hr != 0 {
		comCall(dev, diRelease)
		comCall(di, diRelease)
		return nil, fmt.Errorf("DirectInput DIPROP_GUIDANDPATH failed: 0x%x", hr)
	}
	path := syscall.UTF16ToString(property.Path[:])
	// DirectInput has now completed its sole job for this backend: binding the
	// instance GUID to the exact HID collection. Release it before opening the
	// native path so EZFRD cannot retain a competing device handle.
	comCall(dev, diRelease)
	comCall(di, diRelease)
	low := strings.ToLower(path)
	if path == "" || !strings.Contains(low, pxnVID) || !strings.Contains(low, pxnPID) {
		return nil, fmt.Errorf("DirectInput returned an unexpected HID path for PXN-F19")
	}
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, fmt.Errorf("invalid DirectInput HID path: %w", err)
	}
	h, err := syscall.CreateFile(pathPtr, genericRead|genericWrite, fileShareRead|fileShareWrite, nil, openExisting, fileAttributeNormal, 0)
	if err != nil {
		return nil, fmt.Errorf("open DirectInput-selected PXN-F19 HID path: %w", err)
	}
	m := &hidMotor{handle: h}
	if err := m.write(0, 0); err != nil {
		_ = syscall.CloseHandle(h)
		return nil, fmt.Errorf("initialize PXN-F19 motor output: %w", err)
	}
	return m, nil
}

func (m *hidMotor) write(left, right byte) error {
	if m == nil || m.handle == syscall.InvalidHandle {
		return fmt.Errorf("invalid HID handle")
	}
	report := []byte{0x00, 0x05, 0x01, left, right, 0x00, 0x00, 0x00}
	var written uint32
	if err := syscall.WriteFile(m.handle, report, &written, nil); err != nil {
		return err
	}
	if written != uint32(len(report)) {
		return fmt.Errorf("short HID write: %d/%d", written, len(report))
	}
	return nil
}

func (m *hidMotor) set(left, right float64) error {
	left = clamp(left, 0, 1)
	right = clamp(right, 0, 1)
	// PXNJoystick's independent motor-preview path writes the two recovered
	// amplitude bytes separately. Preserve both simulator-owned channels.
	return m.write(normalizedByte(left), normalizedByte(right))
}

func (m *hidMotor) healthy() error {
	if m == nil || m.handle == syscall.InvalidHandle {
		return errMotorDisconnected
	}
	attributes := hidAttributes{Size: uint32(unsafe.Sizeof(hidAttributes{}))}
	result, _, callErr := syscall.NewLazyDLL("hid.dll").NewProc("HidD_GetAttributes").Call(
		uintptr(m.handle), uintptr(unsafe.Pointer(&attributes)),
	)
	if result == 0 {
		return fmt.Errorf("HidD_GetAttributes: %w", callErr)
	}
	if attributes.VendorID != 0x11ff || attributes.ProductID != 0x0847 {
		return fmt.Errorf("unexpected HID identity %04x/%04x", attributes.VendorID, attributes.ProductID)
	}
	return nil
}

func (m *hidMotor) close() {
	if m == nil || m.handle == syscall.InvalidHandle {
		return
	}
	_ = m.write(0, 0)
	_ = syscall.CloseHandle(m.handle)
	m.handle = syscall.InvalidHandle
}

var selectedDevice *diDeviceInstance
var fallbackDevice *diDeviceInstance

func enumCallback(instance uintptr, _ uintptr) uintptr {
	d := (*diDeviceInstance)(unsafe.Pointer(instance))
	instanceName := syscall.UTF16ToString(d.InstanceName[:])
	productName := syscall.UTF16ToString(d.ProductName[:])
	low := strings.ToLower(instanceName + " " + productName)
	if strings.Contains(low, "pxn") || strings.Contains(low, "f19") || strings.Contains(low, "2119") {
		copyD := *d
		selectedDevice = &copyD
		return 0 // DIENUM_STOP
	}
	devType := d.DevType & 0xff
	if fallbackDevice == nil && (devType == diDevTypeJoystick || devType == diDevTypeGamepad) {
		copyD := *d
		fallbackDevice = &copyD
	}
	return diEnumContinue
}
func newFFB() (*ffb, error) {
	dll := syscall.NewLazyDLL("dinput8.dll")
	if err := dll.Load(); err != nil {
		return nil, fmt.Errorf("load dinput8.dll: %w", err)
	}
	proc := dll.NewProc("DirectInput8Create")
	var di uintptr
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getModule := kernel32.NewProc("GetModuleHandleW")
	module, _, _ := getModule.Call(0)
	hr, _, _ := proc.Call(module, 0x0800, uintptr(unsafe.Pointer(&iidDirectInput8)), uintptr(unsafe.Pointer(&di)), 0)
	if hr != 0 || di == 0 {
		return nil, fmt.Errorf("DirectInput8Create failed: 0x%x", hr)
	}
	selectedDevice = nil
	fallbackDevice = nil
	cb := syscall.NewCallback(enumCallback)
	// IDirectInput8::EnumDevices(dwDevType, callback, context, flags).
	// Keep the callback in the second argument.  The previous extra zero in
	// front of cb made DirectInput receive a NULL callback, so enumeration
	// could never return the PXN device.
	hr = comCall(di, diEnumDevice, diDevClassAll, cb, 0, diedflAttachedOnly|diedflForceFeedback)
	if selectedDevice == nil {
		selectedDevice = fallbackDevice
	}
	if hr != 0 || selectedDevice == nil {
		comCall(di, diRelease)
		return nil, fmt.Errorf("no attached force-feedback DirectInput device (EnumDevices HRESULT 0x%x)", hr)
	}
	var dev uintptr
	hr = comCall(di, diCreateDevice, uintptr(unsafe.Pointer(&selectedDevice.Guid)), uintptr(unsafe.Pointer(&dev)), 0)
	if hr != 0 || dev == 0 {
		comCall(di, diRelease)
		return nil, fmt.Errorf("CreateDevice failed: 0x%x", hr)
	}
	// DirectInput requires a data format before Acquire.  We only consume
	// force-feedback effects, so a compact absolute X/Y format is sufficient;
	// the offsets also match the axes referenced by the DIEFFECT below.
	objects := [2]diObjectDataFormat{
		{Guid: uintptr(unsafe.Pointer(&guidXAxis)), Offset: diJoofsX, Type: didftAbsAxis | didftAnyInstance},
		{Guid: uintptr(unsafe.Pointer(&guidYAxis)), Offset: diJoofsY, Type: didftAbsAxis | didftAnyInstance},
	}
	dataFormat := diDataFormat{
		Size:       uint32(unsafe.Sizeof(diDataFormat{})),
		ObjectSize: uint32(unsafe.Sizeof(diObjectDataFormat{})),
		Flags:      didfAbsAxis,
		DataSize:   8,
		Objects:    uint32(len(objects)),
		ObjectData: uintptr(unsafe.Pointer(&objects[0])),
	}
	hr = comCall(dev, diSetDataFormat, uintptr(unsafe.Pointer(&dataFormat)))
	if hr != 0 {
		comCall(dev, diRelease)
		comCall(di, diRelease)
		return nil, fmt.Errorf("SetDataFormat failed: 0x%x", hr)
	}
	// The desktop belongs to the shell and the console HWND belongs to
	// conhost.exe, while this PXN DirectInput stack also rejects NULL.  Create
	// a hidden top-level window that genuinely belongs to this process.
	hwnd, windowErr := createHiddenWindow(module)
	if windowErr != nil {
		comCall(dev, diRelease)
		comCall(di, diRelease)
		return nil, windowErr
	}
	hr = comCall(dev, diSetCooperative, hwnd, disclExclusive|disclBackground)
	if hr != 0 {
		comCall(dev, diRelease)
		comCall(di, diRelease)
		return nil, fmt.Errorf("SetCooperativeLevel failed: 0x%x", hr)
	}
	hr = comCall(dev, diAcquire)
	if hr != 0 && uint32(hr) != 0x8007001e { // DIERR_INPUTLOST can be transient
		comCall(dev, diRelease)
		comCall(di, diRelease)
		return nil, fmt.Errorf("Acquire failed: 0x%x", hr)
	}
	f := &ffb{dinput: di, device: dev, axes: [2]uint32{diJoofsX, diJoofsY}, directions: [2]int32{10000, 0}}
	if err := f.createEffect(); err != nil {
		f.close()
		return nil, err
	}
	return f, nil
}
func (f *ffb) createEffect() error {
	// EZFRD effect ID 0 is ConstantForce.  Prefer it because the recovered
	// driver maps its magnitude directly to the motor amplitude; periodic
	// effects are sampled by EZFRD's much slower 150 ms multimedia timer and
	// can alias badly during a short hardware test.
	f.constant = diConstant{}
	e := diEffect{Size: uint32(unsafe.Sizeof(diEffect{})), Flags: dieffCartesian | dieffObjectOffsets, Duration: diInfinite, Gain: diFfNominalMax, TriggerBtn: diInfinite, TriggerRep: diInfinite, Axes: 2, RGDAxes: uintptr(unsafe.Pointer(&f.axes[0])), Directions: uintptr(unsafe.Pointer(&f.directions[0])), TypeSize: uint32(unsafe.Sizeof(f.constant)), TypeParams: uintptr(unsafe.Pointer(&f.constant))}
	var out uintptr
	hr := comCall(f.device, diCreateEffect, uintptr(unsafe.Pointer(&guidConstant)), uintptr(unsafe.Pointer(&e)), uintptr(unsafe.Pointer(&out)), 0)
	if hr == 0 && out != 0 {
		startHR := comCall(out, diStartEffect, 1, 0)
		if startHR == 0 {
			f.effect, f.usingPeriodic = out, false
			fmt.Println("DirectInput effect: ConstantForce")
			return nil
		}
		comCall(out, diRelease)
		out = 0
		hr = startHR
	}
	constantHR := hr

	f.periodic = diPeriodic{Period: 28000}
	e.TypeSize = uint32(unsafe.Sizeof(f.periodic))
	e.TypeParams = uintptr(unsafe.Pointer(&f.periodic))
	hr = comCall(f.device, diCreateEffect, uintptr(unsafe.Pointer(&guidSine)), uintptr(unsafe.Pointer(&e)), uintptr(unsafe.Pointer(&out)), 0)
	if hr != 0 || out == 0 {
		if uint32(hr) == 0x80040154 {
			return fmt.Errorf("EZFRD COM class is not registered (0x%x); run register-ezfrd.cmd as Administrator", hr)
		}
		return fmt.Errorf("PXN force-feedback effects unavailable (ConstantForce 0x%x, Sine 0x%x)", constantHR, hr)
	}
	startHR := comCall(out, diStartEffect, 1, 0)
	if startHR != 0 {
		comCall(out, diRelease)
		return fmt.Errorf("PXN Sine effect start failed: 0x%x (ConstantForce 0x%x)", startHR, constantHR)
	}
	f.effect, f.usingPeriodic = out, true
	fmt.Println("DirectInput effect: Sine (ConstantForce unavailable)")
	return nil
}

func (f *ffb) reportSetResult(hr uintptr) error {
	code := uint32(hr)
	if code == 0 {
		f.lastSetHR = 0
		return nil
	}
	if code != f.lastSetHR {
		fmt.Fprintf(os.Stderr, "DirectInput SetParameters failed: 0x%x\n", code)
		f.lastSetHR = code
	}
	return fmt.Errorf("DirectInput SetParameters failed: 0x%x", code)
}

func (f *ffb) set(left, right float64) error {
	if f == nil || f.effect == 0 {
		return fmt.Errorf("DirectInput effect is unavailable")
	}
	// EZFRD reduces both motor bytes to a single evaluated amplitude. Keep this
	// backend only for diagnostics and use the greater requested channel.
	v := math.Max(left, right)
	v = clamp(v, 0, 1)
	if f.usingPeriodic {
		f.periodic.Magnitude = uint32(v * diFfNominalMax)
		f.periodic.Phase = 0
		f.directions[0] = diFfNominalMax
		f.directions[1] = 0
		e := diEffect{Size: uint32(unsafe.Sizeof(diEffect{})), Flags: dieffCartesian | dieffObjectOffsets, Duration: diInfinite, Gain: diFfNominalMax, TriggerBtn: diInfinite, TriggerRep: diInfinite, Axes: 2, RGDAxes: uintptr(unsafe.Pointer(&f.axes[0])), Directions: uintptr(unsafe.Pointer(&f.directions[0])), TypeSize: uint32(unsafe.Sizeof(f.periodic)), TypeParams: uintptr(unsafe.Pointer(&f.periodic))}
		return f.reportSetResult(comCall(f.effect, diSetEffectParams, uintptr(unsafe.Pointer(&e)), diepTypespecific))
	} else {
		// Constant force has a signed range, but zero vibration must mean
		// zero force rather than the negative end stop.
		f.constant.Magnitude = int32(v * diFfNominalMax)
		e := diEffect{Size: uint32(unsafe.Sizeof(diEffect{})), Flags: dieffCartesian | dieffObjectOffsets, Duration: diInfinite, Gain: diFfNominalMax, TriggerBtn: diInfinite, TriggerRep: diInfinite, Axes: 2, RGDAxes: uintptr(unsafe.Pointer(&f.axes[0])), Directions: uintptr(unsafe.Pointer(&f.directions[0])), TypeSize: uint32(unsafe.Sizeof(f.constant)), TypeParams: uintptr(unsafe.Pointer(&f.constant))}
		return f.reportSetResult(comCall(f.effect, diSetEffectParams, uintptr(unsafe.Pointer(&e)), diepTypespecific))
	}
}

func (f *ffb) healthy() error {
	if f == nil || f.device == 0 || f.effect == 0 {
		return errMotorDisconnected
	}
	return nil
}
func (f *ffb) close() {
	if f == nil {
		return
	}
	if f.effect != 0 {
		comCall(f.effect, diRelease)
	}
	if f.device != 0 {
		comCall(f.device, diRelease)
	}
	if f.dinput != 0 {
		comCall(f.dinput, diRelease)
	}
}

type simRecv struct{ Size, Version, ID uint32 }

const simRecvSimobjectData = 8
const simPeriodSimFrame = 2

type msfs struct{ handle, addDef, reqData, getNext, close uintptr }

func cstr(s string) uintptr { b := append([]byte(s), 0); return uintptr(unsafe.Pointer(&b[0])) }

func createHiddenWindow(module uintptr) (uintptr, error) {
	user32 := syscall.NewLazyDLL("user32.dll")
	createWindow := user32.NewProc("CreateWindowExW")
	className, err := syscall.UTF16PtrFromString("STATIC")
	if err != nil {
		return 0, err
	}
	windowName, err := syscall.UTF16PtrFromString("PXN 2119 FFB Bridge")
	if err != nil {
		return 0, err
	}
	// STATIC is a predefined system class.  With no WS_VISIBLE bit and no
	// parent this creates a hidden top-level window owned by this process.
	hwnd, _, callErr := createWindow.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(windowName)),
		0,
		0, 0, 1, 1,
		0, 0, module, 0,
	)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowExW failed: %v", callErr)
	}
	return hwnd, nil
}

// installConsoleStopHandler also covers closing the console window, logoff,
// and shutdown. For those events Windows may terminate the process as soon as
// the handler returns, so the handler waits until main has sent the stop report.
func installConsoleStopHandler() (<-chan struct{}, func()) {
	events := make(chan struct{}, 1)
	acknowledged := make(chan struct{})
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setHandler := kernel32.NewProc("SetConsoleCtrlHandler")
	callback := syscall.NewCallback(func(event uintptr) uintptr {
		switch event {
		case 0, 1, 2, 5, 6: // CTRL_C/BREAK/CLOSE/LOGOFF/SHUTDOWN_EVENT
			select {
			case events <- struct{}{}:
			default:
			}
			<-acknowledged
			return 1
		default:
			return 0
		}
	})
	ok, _, _ := setHandler.Call(callback, 1)
	if ok == 0 {
		return nil, func() { runtime.KeepAlive(callback) }
	}
	return events, func() {
		close(acknowledged)
		setHandler.Call(callback, 0)
		runtime.KeepAlive(callback)
	}
}

func openMSFS() (*msfs, error) {
	dll := syscall.NewLazyDLL("SimConnect.dll")
	if err := dll.Load(); err != nil {
		return nil, err
	}
	get := func(n string) uintptr { return dll.NewProc(n).Addr() }
	m := &msfs{addDef: get("SimConnect_AddToDataDefinition"), reqData: get("SimConnect_RequestDataOnSimObject"), getNext: get("SimConnect_GetNextDispatch"), close: get("SimConnect_Close")}
	open := dll.NewProc("SimConnect_Open")
	hr, _, _ := open.Call(uintptr(unsafe.Pointer(&m.handle)), cstr("PXN 2119 FFB Bridge"), 0, 0, 0, 0)
	if hr != 0 || m.handle == 0 {
		return nil, fmt.Errorf("SimConnect_Open failed: 0x%x", hr)
	}
	defs := []struct{ n, u string }{{"ACCELERATION BODY X", "feet per second squared"}, {"ACCELERATION BODY Y", "feet per second squared"}, {"ACCELERATION BODY Z", "feet per second squared"}, {"AIRSPEED INDICATED", "knots"}, {"VERTICAL SPEED", "feet per minute"}, {"SIM ON GROUND", "Bool"}, {"STALL WARNING", "Bool"}}
	for i, d := range defs {
		hr, _, _ = syscall.SyscallN(m.addDef, m.handle, 0, cstr(d.n), cstr(d.u), 4, 0, uintptr(i))
		if hr != 0 {
			fmt.Fprintf(os.Stderr, "warning: SimConnect definition %s failed: 0x%x\n", d.n, hr)
		}
	}
	hr, _, _ = syscall.SyscallN(m.reqData, m.handle, 0, 0, 0, simPeriodSimFrame, 0, 0, 0, 0, 0)
	if hr != 0 {
		m.Close()
		return nil, fmt.Errorf("SimConnect_RequestDataOnSimObject failed: 0x%x", hr)
	}
	return m, nil
}
func (m *msfs) Poll() (telemetry, bool) {
	if m == nil || m.handle == 0 {
		return telemetry{}, false
	}
	var p uintptr
	var size uint32
	hr, _, _ := syscall.SyscallN(m.getNext, m.handle, uintptr(unsafe.Pointer(&p)), uintptr(unsafe.Pointer(&size)))
	if hr != 0 || p == 0 || size < 40 {
		return telemetry{}, false
	}
	if (*(*simRecv)(unsafe.Pointer(p))).ID != simRecvSimobjectData {
		return telemetry{}, false
	}
	v := (*[7]float64)(unsafe.Pointer(p + 40))
	return telemetry{GX: v[0] / 32.174, GY: v[1] / 32.174, GZ: v[2] / 32.174, Airspeed: v[3], Vertical: v[4], Ground: v[5], Stall: v[6]}, true
}
func (m *msfs) Close() {
	if m != nil && m.handle != 0 {
		syscall.SyscallN(m.close, m.handle)
		m.handle = 0
	}
}

type xplane struct {
	conn *net.UDPConn
	addr *net.UDPAddr
	last telemetry
}

var xrefs = []string{"sim/flightmodel/forces/g_nrml", "sim/flightmodel/forces/g_side", "sim/flightmodel/forces/g_axil", "sim/flightmodel/position/true_airspeed", "sim/flightmodel/position/vh_ind_fpm", "sim/flightmodel/failures/onground_any", "sim/flightmodel/forces/stallwarning"}

func openXPlane(port int) (*xplane, error) {
	addr, e := net.ResolveUDPAddr("udp", fmt.Sprintf("127.0.0.1:%d", port))
	if e != nil {
		return nil, e
	}
	c, e := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if e != nil {
		return nil, e
	}
	x := &xplane{conn: c, addr: addr}
	for i, d := range xrefs {
		x.request(i, d)
	}
	return x, nil
}
func (x *xplane) request(id int, d string) {
	b := make([]byte, 413)
	copy(b, []byte("RREF\x00"))
	binary.LittleEndian.PutUint32(b[5:9], 20)
	binary.LittleEndian.PutUint32(b[9:13], uint32(id))
	copy(b[13:], []byte(d))
	_, _ = x.conn.WriteToUDP(b, x.addr)
}
func (x *xplane) Poll() telemetry {
	_ = x.conn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	buf := make([]byte, 1024)
	for {
		n, _, e := x.conn.ReadFromUDP(buf)
		if e != nil {
			break
		}
		if n < 13 || string(buf[:5]) != "RREF," {
			continue
		}
		id := int(binary.LittleEndian.Uint32(buf[5:9]))
		val := float64(math.Float32frombits(binary.LittleEndian.Uint32(buf[9:13])))
		if id < 0 || id >= len(xrefs) {
			continue
		}
		switch id {
		case 0:
			x.last.GZ = val
		case 1:
			x.last.GY = val
		case 2:
			x.last.GX = val
		case 3:
			x.last.Airspeed = val * 1.943844
		case 4:
			x.last.Vertical = val
		case 5:
			x.last.Ground = val
		case 6:
			x.last.Stall = val
		}
	}
	return x.last
}
func (x *xplane) Close() {
	if x != nil && x.conn != nil {
		x.conn.Close()
	}
}
func vibration(t telemetry) (float64, float64) {
	impact := clamp(math.Abs(t.GZ-1)/2.5, 0, 1)
	trans := clamp((math.Abs(t.GX)+math.Abs(t.GY))/2, 0, 1)
	ground := clamp(t.Ground, 0, 1) * clamp(t.Airspeed/180, 0, 1) * .32
	stall := clamp(t.Stall, 0, 1) * .75
	return clamp(impact*.65+trans*.25+ground+stall, 0, 1), clamp(t.GY/2, -1, 1)
}
func main() {
	mode := flag.String("mode", "service", "service, test, motor-test, calibrate, or legacy auto/msfs/xplane")
	outputMode := flag.String("output", "hidpath", "motor output: hidpath or directinput")
	gain := flag.Float64("gain", 2.0, "simulator vibration gain before the safe 0..1 clamp")
	testOn := flag.Duration("test-on", 1500*time.Millisecond, "test-mode vibration duration per cycle")
	testCycle := flag.Duration("test-cycle", 3*time.Second, "test-mode cycle duration")
	testChannel := flag.String("channel", "both", "motor-test channel: stick, throttle, or both (left/right aliases supported)")
	testAmplitude := flag.Int("amplitude", 255, "motor-test amplitude: 0..255")
	testDuration := flag.Duration("duration", 3*time.Second, "motor-test pulse duration")
	configPathFlag := flag.String("config", "", "calibration JSON path (default: LocalAppData)")
	rawOutput := flag.Bool("raw", false, "motor-test: bypass saved calibration")
	resetCalibration := flag.Bool("reset", false, "calibrate mode: restore linear defaults")
	serviceListen := flag.String("service-listen", "127.0.0.1:21190", "localhost simulator-command UDP endpoint")
	statusInterval := flag.Duration("status-interval", time.Second, "service status display interval; 0 disables periodic status")
	msfsAdapter := flag.String("msfs-adapter", "auto", "service mode MSFS adapter: auto, off, or executable path")
	msfsTelemetry := flag.Bool("msfs-telemetry", true, "MSFS adapter: generate vibration from simulator telemetry")
	msfsTelemetryGain := flag.Float64("msfs-telemetry-gain", 1.25, "MSFS adapter telemetry vibration gain: 0..4")
	stickMaximum := flag.Int("stick-max", defaultStickMaximum, "final joystick-base motor maximum: 0..255")
	throttleMaximum := flag.Int("throttle-max", defaultThrottleMaximum, "final throttle motor maximum: 0..255")
	motorBudget := flag.Int("motor-budget", defaultMotorBudget, "maximum combined stick+throttle output: 1..510")
	motorSlew := flag.Int("motor-slew", defaultMotorSlew, "maximum per-channel rising step per output update: 1..255")
	port := flag.Int("xplane-port", 49000, "X-Plane UDP port")
	interval := flag.Duration("interval", 30*time.Millisecond, "output update interval")
	flag.Parse()
	*mode = strings.ToLower(*mode)
	*outputMode = strings.ToLower(*outputMode)
	*testChannel = strings.ToLower(*testChannel)
	if *testChannel == "left" {
		*testChannel = "stick"
	}
	if *testChannel == "right" {
		*testChannel = "throttle"
	}
	switch *mode {
	case "service", "test", "motor-test", "calibrate", "auto", "msfs", "xplane":
	default:
		fmt.Fprintf(os.Stderr, "unknown --mode %q (use service, test, motor-test, calibrate, auto, msfs, or xplane)\n", *mode)
		os.Exit(2)
	}
	if *gain < 0 || *testOn < 0 || *testCycle <= 0 || *testOn > *testCycle || *interval <= 0 || *statusInterval < 0 || *msfsTelemetryGain < 0 || *msfsTelemetryGain > 4 {
		fmt.Fprintln(os.Stderr, "invalid gain/test timing: require gain >= 0 and 0 <= test-on <= test-cycle")
		os.Exit(2)
	}
	safetyConfig := motorSafetyConfig{
		StickMaximum:    *stickMaximum,
		ThrottleMaximum: *throttleMaximum,
		TotalBudget:     *motorBudget,
		MotorSlew:       *motorSlew,
	}
	if err := safetyConfig.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "invalid motor safety setting: %v\n", err)
		os.Exit(2)
	}
	if *outputMode != "hidpath" && *outputMode != "directinput" {
		fmt.Fprintf(os.Stderr, "unknown --output %q (use directinput or hidpath)\n", *outputMode)
		os.Exit(2)
	}
	if *rawOutput && *mode != "motor-test" {
		fmt.Fprintln(os.Stderr, "--raw is only valid with --mode motor-test")
		os.Exit(2)
	}
	if *resetCalibration && *mode != "calibrate" {
		fmt.Fprintln(os.Stderr, "--reset is only valid with --mode calibrate")
		os.Exit(2)
	}
	if (*mode == "motor-test" || *mode == "calibrate") && *outputMode != "hidpath" {
		fmt.Fprintln(os.Stderr, "motor-test and calibrate require --output hidpath for independent motor access")
		os.Exit(2)
	}
	if *mode == "motor-test" {
		if *testChannel != "stick" && *testChannel != "throttle" && *testChannel != "both" {
			fmt.Fprintln(os.Stderr, "--channel must be stick, throttle, or both")
			os.Exit(2)
		}
		if *testAmplitude < 0 || *testAmplitude > 255 || *testDuration <= 0 || *testDuration > 30*time.Second {
			fmt.Fprintln(os.Stderr, "motor-test requires amplitude 0..255 and duration within 0..30s")
			os.Exit(2)
		}
	}

	configPath := *configPathFlag
	if configPath == "" {
		var err error
		configPath, err = defaultCalibrationPath()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	}
	config, configErr := loadCalibration(configPath)
	if configErr != nil {
		fmt.Fprintf(os.Stderr, "warning: calibration %s is invalid (%v); using linear defaults\n", configPath, configErr)
	}
	if *mode == "calibrate" && *resetCalibration {
		if err := saveCalibration(configPath, defaultCalibration()); err != nil {
			fmt.Fprintf(os.Stderr, "reset calibration: %v\n", err)
			os.Exit(2)
		}
		fmt.Printf("Calibration reset to linear defaults: %s\n", configPath)
		return
	}

	consoleStop, removeConsoleHandler := installConsoleStopHandler()
	defer removeConsoleHandler()
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	stopRequested := make(chan struct{})
	go func() {
		select {
		case <-interrupt:
		case <-consoleStop:
		}
		close(stopRequested)
	}()

	var e error
	var service *motorService
	var reconnecting *reconnectingMotor
	if *mode == "service" {
		service, e = openMotorService(*serviceListen)
		if e != nil {
			fmt.Fprintln(os.Stderr, e)
			return
		}
		defer service.close()
		fmt.Printf("PXN motor service listening on udp://%s (PXNF/1)\n", *serviceListen)
		fmt.Println("Vibration content is controlled by the simulator adapter")
		fmt.Printf("Motor safety: stick-max=%d throttle-max=%d budget=%d slew=%d/update\n",
			safetyConfig.StickMaximum, safetyConfig.ThrottleMaximum,
			safetyConfig.TotalBudget, safetyConfig.MotorSlew)
		reconnecting = newReconnectingMotor(func() (motorOutput, string, error) {
			raw, label, err := openMotorOutput(*outputMode)
			if err != nil {
				return nil, "", err
			}
			return raw, label, nil
		}, time.Duration(config.ReconnectIntervalMS)*time.Millisecond)
		defer reconnecting.close()
		adapterSupervisor, supervisorErr := newMSFSAdapterSupervisor(
			*msfsAdapter, *serviceListen, *msfsTelemetry, *msfsTelemetryGain,
		)
		if supervisorErr != nil {
			fmt.Fprintf(os.Stderr, "MSFS adapter auto-start disabled: %v\n", supervisorErr)
		} else if adapterSupervisor != nil {
			adapterDone := adapterSupervisor.start(stopRequested)
			defer func() { <-adapterDone }()
			fmt.Printf("MSFS adapter supervision enabled: %s\n", adapterSupervisor.path)
		}
	} else {
		raw, label, err := openMotorOutput(*outputMode)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(label)
		if *mode == "calibrate" {
			defer raw.close()
			if err := runCalibration(raw, configPath, stopRequested); err != nil {
				fmt.Fprintf(os.Stderr, "Calibration failed: %v\n", err)
				return
			}
			return
		}
		var output motorOutput = &calibratedMotor{raw: raw, config: config}
		if *mode == "motor-test" && *rawOutput {
			output = raw
		}
		defer output.close()
		if *mode == "motor-test" {
			command := motorCommand{}
			if *testChannel == "stick" || *testChannel == "both" {
				command.Left = byte(*testAmplitude)
			}
			if *testChannel == "throttle" || *testChannel == "both" {
				command.Right = byte(*testAmplitude)
			}
			fmt.Printf("Motor test: channel=%s amplitude=%d duration=%s raw=%t\n", *testChannel, *testAmplitude, testDuration.String(), *rawOutput)
			if err := output.set(normalizedFloat(command.Left), normalizedFloat(command.Right)); err != nil {
				fmt.Fprintf(os.Stderr, "Motor test failed: %v\n", err)
				return
			}
			timer := time.NewTimer(*testDuration)
			select {
			case <-stopRequested:
				if !timer.Stop() {
					<-timer.C
				}
			case <-timer.C:
			}
			fmt.Println("Stopping PXN-F19 vibration...")
			return
		}
		defer func() {
			_ = output.set(0, 0)
		}()

		var m *msfs
		var x *xplane
		if *mode == "auto" || *mode == "msfs" {
			m, _ = openMSFS()
		}
		if m != nil {
			defer m.Close()
			fmt.Println("MSFS SimConnect connected")
		}
		if (*mode == "auto" && m == nil) || *mode == "xplane" {
			x, e = openXPlane(*port)
			if e != nil && *mode == "xplane" {
				fmt.Fprintln(os.Stderr, e)
				return
			}
		}
		if x != nil {
			defer x.Close()
			fmt.Println("X-Plane RREF listener active")
		}
		runOutputLoop(stopRequested, *mode, output, nil, nil, m, x, config, safetyConfig, *gain, *testOn, *testCycle, *interval, *statusInterval)
		return
	}
	runOutputLoop(stopRequested, *mode, nil, reconnecting, service, nil, nil, config, safetyConfig, *gain, *testOn, *testCycle, *interval, *statusInterval)
}

func runOutputLoop(stopRequested <-chan struct{}, mode string, output motorOutput, reconnecting *reconnectingMotor, service *motorService, m *msfs, x *xplane, config calibrationConfig, safetyConfig motorSafetyConfig, gain float64, testOn, testCycle, interval, statusInterval time.Duration) {
	tick := time.NewTicker(interval)
	defer tick.Stop()
	start := time.Now()
	lastConnectionError := ""
	everConnected := false
	currentApplied := motorCommand{}
	safety := newMotorSafetyLimiter(safetyConfig)
	nextSafetyNotice := time.Time{}
	nextStatus := time.Now()
	for {
		select {
		case <-stopRequested:
			fmt.Println("Stopping PXN-F19 vibration...")
			return
		case <-tick.C:
		}
		now := time.Now()
		if reconnecting != nil && reconnecting.connected() {
			if err := reconnecting.healthCheck(now); err != nil {
				currentApplied = motorCommand{}
				safety.reset()
				fmt.Fprintf(os.Stderr, "PXN-F19 disconnected: %v; reconnect scheduled\n", err)
				lastConnectionError = err.Error()
			}
		}
		if reconnecting != nil && !reconnecting.connected() {
			currentApplied = motorCommand{}
			safety.reset()
			// Drain and acknowledge commands while disconnected before attempting
			// a reconnect. This guarantees that an old SET is never replayed when
			// the device becomes available again.
			_, _ = service.poll(now)
			connected, label, err := reconnecting.tick(now)
			if err != nil {
				message := err.Error()
				if message != lastConnectionError {
					fmt.Fprintf(os.Stderr, "PXN-F19 unavailable; waiting for reconnect: %v\n", err)
					lastConnectionError = message
				}
			} else if connected {
				if everConnected || lastConnectionError != "" {
					fmt.Printf("PXN-F19 reconnected: %s\n", label)
				} else {
					fmt.Println(label)
				}
				everConnected = true
				lastConnectionError = ""
			}
			printServiceStatus(now, &nextStatus, statusInterval, service, reconnecting, currentApplied)
			continue
		}
		var t telemetry
		var ok bool
		switch {
		case reconnecting != nil:
			command, changed := service.poll(now)
			if changed {
				calibrated := config.apply(command)
				constrained, limited := safety.setTarget(calibrated)
				if limited && !now.Before(nextSafetyNotice) {
					fmt.Printf("Motor safety limit: requested stick=%d throttle=%d applied-target stick=%d throttle=%d\n",
						calibrated.Left, calibrated.Right, constrained.Left, constrained.Right)
					nextSafetyNotice = now.Add(time.Second)
				}
			}
			applied, outputChanged := safety.advance()
			if outputChanged {
				if err := reconnecting.apply(now, normalizedFloat(applied.Left), normalizedFloat(applied.Right)); err != nil {
					currentApplied = motorCommand{}
					safety.reset()
					if !errors.Is(err, errMotorDisconnected) {
						fmt.Fprintf(os.Stderr, "PXN-F19 output failed; reconnect scheduled: %v\n", err)
					}
				} else {
					currentApplied = applied
				}
			}
			printServiceStatus(now, &nextStatus, statusInterval, service, reconnecting, currentApplied)
			continue
		case mode == "test":
			// The native output is already at the hardware's full byte amplitude.
			// A longer pulse gives the small eccentric motors time to spin up and
			// makes the real-device result easier to distinguish.
			active := time.Duration(time.Since(start).Nanoseconds()%testCycle.Nanoseconds()) < testOn
			if active {
				if err := output.set(1, 1); err != nil {
					fmt.Fprintf(os.Stderr, "PXN-F19 output failed: %v\n", err)
				}
			} else {
				if err := output.set(0, 0); err != nil {
					fmt.Fprintf(os.Stderr, "PXN-F19 output failed: %v\n", err)
				}
			}
			continue
		case m != nil:
			t, ok = m.Poll()
		case x != nil:
			t = x.Poll()
			ok = true
		}
		if ok {
			a, _ := vibration(t)
			amplitude := clamp(a*gain, 0, 1)
			if err := output.set(amplitude, amplitude); err != nil {
				fmt.Fprintf(os.Stderr, "PXN-F19 output failed: %v\n", err)
			}
		}
	}
}

func printServiceStatus(now time.Time, next *time.Time, interval time.Duration, service *motorService, motor *reconnectingMotor, command motorCommand) {
	if interval <= 0 || now.Before(*next) {
		return
	}
	*next = now.Add(interval)
	device := "disconnected"
	if motor != nil && motor.connected() {
		device = "online"
	}
	simulator := service.snapshot()
	fmt.Printf("Status: device=%s adapter=%s simulator=%s stick=%d throttle=%d\n",
		device, simulator.Adapter, simulator.State, command.Left, command.Right)
}
