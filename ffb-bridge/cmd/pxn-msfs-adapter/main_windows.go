//go:build windows

package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"pxn2119-ffb-bridge/internal/winproc"
)

const (
	clientDataID          = 0x2119
	definitionID          = 0x2119
	requestID             = 0x2119
	telemetryDefinitionID = 0x2130
	telemetryRequestID    = 0x2131
	pauseEventID          = 0x2120
	simStartEventID       = 0x2121
	simStopEventID        = 0x2122

	recvIDException     = 1
	recvIDOpen          = 2
	recvIDQuit          = 3
	recvIDEvent         = 4
	recvIDSimObjectData = 8
	recvIDClientData    = 16

	clientDataPeriodOnSet     = 3
	clientDataChanged         = 1
	simConnectPeriodSimFrame  = 3
	simConnectDataTypeFloat64 = 4
	simConnectUnused          = 0xffffffff
	heartbeatInterval         = 500 * time.Millisecond
	processCheckInterval      = 500 * time.Millisecond
	telemetryCommandDuration  = 150 * time.Millisecond
)

type simConnectAPI struct {
	dll                       *syscall.DLL
	open                      *syscall.Proc
	close                     *syscall.Proc
	callDispatch              *syscall.Proc
	mapClientDataNameToID     *syscall.Proc
	createClientData          *syscall.Proc
	addToClientDataDefinition *syscall.Proc
	requestClientData         *syscall.Proc
	subscribeToSystemEvent    *syscall.Proc
	addToDataDefinition       *syscall.Proc
	requestDataOnSimObject    *syscall.Proc
}

func loadSimConnect() (*simConnectAPI, string, error) {
	executable, _ := os.Executable()
	candidates := []string{
		filepath.Join(filepath.Dir(executable), "SimConnect.dll"),
		"SimConnect.dll",
	}
	if sdk := strings.TrimSpace(os.Getenv("MSFS_SDK")); sdk != "" {
		candidates = append(candidates,
			filepath.Join(sdk, "SimConnect SDK", "lib", "SimConnect.dll"),
			filepath.Join(sdk, "SimConnect SDK", "lib", "static", "SimConnect.dll"),
		)
	}
	seen := make(map[string]bool)
	var failures []string
	for _, candidate := range candidates {
		key := strings.ToLower(candidate)
		if seen[key] {
			continue
		}
		seen[key] = true
		dll, err := syscall.LoadDLL(candidate)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", candidate, err))
			continue
		}
		api := &simConnectAPI{dll: dll}
		procedures := []struct {
			name string
			dest **syscall.Proc
		}{
			{"SimConnect_Open", &api.open},
			{"SimConnect_Close", &api.close},
			{"SimConnect_CallDispatch", &api.callDispatch},
			{"SimConnect_MapClientDataNameToID", &api.mapClientDataNameToID},
			{"SimConnect_CreateClientData", &api.createClientData},
			{"SimConnect_AddToClientDataDefinition", &api.addToClientDataDefinition},
			{"SimConnect_RequestClientData", &api.requestClientData},
			{"SimConnect_SubscribeToSystemEvent", &api.subscribeToSystemEvent},
			{"SimConnect_AddToDataDefinition", &api.addToDataDefinition},
			{"SimConnect_RequestDataOnSimObject", &api.requestDataOnSimObject},
		}
		valid := true
		for _, procedure := range procedures {
			proc, findErr := dll.FindProc(procedure.name)
			if findErr != nil {
				failures = append(failures, fmt.Sprintf("%s:%s: %v", candidate, procedure.name, findErr))
				valid = false
				break
			}
			*procedure.dest = proc
		}
		if valid {
			return api, candidate, nil
		}
		_ = dll.Release()
	}
	return nil, "", fmt.Errorf("load SimConnect.dll (%s)", strings.Join(failures, "; "))
}

func failedHRESULT(result uintptr) bool {
	return int32(uint32(result)) < 0
}

func checkHRESULT(operation string, result uintptr) error {
	if failedHRESULT(result) {
		return fmt.Errorf("%s failed: HRESULT 0x%08X", operation, uint32(result))
	}
	return nil
}

func bytePointer(value string) (*byte, error) {
	return syscall.BytePtrFromString(value)
}

func (api *simConnectAPI) connect() (syscall.Handle, error) {
	name, err := bytePointer("PXN 2119 FFB Adapter")
	if err != nil {
		return 0, err
	}
	var handle syscall.Handle
	result, _, _ := api.open.Call(
		uintptr(unsafe.Pointer(&handle)),
		uintptr(unsafe.Pointer(name)),
		0, 0, 0, 0,
	)
	if err := checkHRESULT("SimConnect_Open", result); err != nil {
		return 0, err
	}
	return handle, nil
}

func (api *simConnectAPI) setup(handle syscall.Handle) error {
	clientName, _ := bytePointer("PXN2119_FFB")
	result, _, _ := api.mapClientDataNameToID.Call(
		uintptr(handle), uintptr(unsafe.Pointer(clientName)), clientDataID,
	)
	if err := checkHRESULT("SimConnect_MapClientDataNameToID", result); err != nil {
		return err
	}

	// Creation is intentionally best-effort: simulator-side content may already
	// own the ClientData area. Mapping, definition, and request must still pass.
	api.createClientData.Call(uintptr(handle), clientDataID, packetSize, 0)
	result, _, _ = api.addToClientDataDefinition.Call(
		uintptr(handle), definitionID, 0, packetSize, 0, simConnectUnused,
	)
	if err := checkHRESULT("SimConnect_AddToClientDataDefinition", result); err != nil {
		return err
	}
	result, _, _ = api.requestClientData.Call(
		uintptr(handle), clientDataID, requestID, definitionID,
		clientDataPeriodOnSet, clientDataChanged, 0, 0, 0,
	)
	if err := checkHRESULT("SimConnect_RequestClientData", result); err != nil {
		return err
	}
	for _, event := range []struct {
		id   uintptr
		name string
	}{
		{pauseEventID, "Pause"},
		{simStartEventID, "SimStart"},
		{simStopEventID, "SimStop"},
	} {
		name, _ := bytePointer(event.name)
		result, _, _ = api.subscribeToSystemEvent.Call(
			uintptr(handle), event.id, uintptr(unsafe.Pointer(name)),
		)
		if err := checkHRESULT("SimConnect_SubscribeToSystemEvent("+event.name+")", result); err != nil {
			return err
		}
	}
	return api.setupTelemetry(handle)
}

var telemetryDefinitions = []struct {
	name string
	unit string
}{
	{"ACCELERATION BODY X", "feet per second squared"},
	{"ACCELERATION BODY Y", "feet per second squared"},
	{"ACCELERATION BODY Z", "feet per second squared"},
	{"G FORCE", "GForce"},
	{"AIRSPEED INDICATED", "knots"},
	{"VERTICAL SPEED", "feet per minute"},
	{"SIM ON GROUND", "Bool"},
	{"STALL WARNING", "Bool"},
	{"GROUND VELOCITY", "knots"},
	{"GENERAL ENG RPM:1", "RPM"},
	{"TURB ENG N1:1", "Percent"},
	{"GENERAL ENG COMBUSTION:1", "Bool"},
}

func (api *simConnectAPI) setupTelemetry(handle syscall.Handle) error {
	for index, definition := range telemetryDefinitions {
		name, _ := bytePointer(definition.name)
		unit, _ := bytePointer(definition.unit)
		result, _, _ := api.addToDataDefinition.Call(
			uintptr(handle), telemetryDefinitionID,
			uintptr(unsafe.Pointer(name)), uintptr(unsafe.Pointer(unit)),
			simConnectDataTypeFloat64, 0, uintptr(index),
		)
		if err := checkHRESULT("SimConnect_AddToDataDefinition("+definition.name+")", result); err != nil {
			return err
		}
	}
	result, _, _ := api.requestDataOnSimObject.Call(
		uintptr(handle), telemetryRequestID, telemetryDefinitionID, 0,
		simConnectPeriodSimFrame, 0, 0, 1, 0,
	)
	return checkHRESULT("SimConnect_RequestDataOnSimObject(telemetry)", result)
}

type recvHeader struct {
	Size    uint32
	Version uint32
	ID      uint32
}

type recvOpen struct {
	recvHeader
	ApplicationName         [256]byte
	ApplicationVersionMajor uint32
	ApplicationVersionMinor uint32
	ApplicationBuildMajor   uint32
	ApplicationBuildMinor   uint32
	SimConnectVersionMajor  uint32
	SimConnectVersionMinor  uint32
	SimConnectBuildMajor    uint32
	SimConnectBuildMinor    uint32
	Reserved1               uint32
	Reserved2               uint32
}

type recvEvent struct {
	recvHeader
	GroupID uint32
	EventID uint32
	Data    uint32
}

type recvClientData struct {
	recvHeader
	RequestID   uint32
	ObjectID    uint32
	DefineID    uint32
	Flags       uint32
	EntryNumber uint32
	OutOf       uint32
	DefineCount uint32
	Data        uint32
}

type recvSimObjectData = recvClientData

func cString(value []byte) string {
	for index, b := range value {
		if b == 0 {
			return string(value[:index])
		}
	}
	return string(value)
}

type adapterSession struct {
	api               *simConnectAPI
	handle            syscall.Handle
	connection        *net.UDPConn
	state             runtimeState
	sequence          uint32
	lastForwarded     time.Time
	running           bool
	exitSent          bool
	telemetryEnabled  bool
	telemetryGain     float64
	telemetryInterval time.Duration
	lastTelemetrySent time.Time
	lastTelemetryLog  time.Time
	telemetryActive   bool
	externalUntil     time.Time
	mixer             telemetryMixer
	telemetrySeen     bool
	lastSimStart      time.Time
	telemetryWarned   bool
	firstPacketSent   time.Time
	serviceAckSeen    bool
	serviceAckWarned  bool
	simConnected      bool
}

var activeSession *adapterSession

func dispatchCallback(data, size, _ uintptr) uintptr {
	if activeSession != nil {
		activeSession.dispatch(data, size)
	}
	return 0
}

var callbackPointer = syscall.NewCallback(dispatchCallback)

func (session *adapterSession) send(input packet, stateFlags uint16) bool {
	decorated, ok := session.state.decorate(input, stateFlags)
	if !ok {
		return false
	}
	if _, err := session.connection.Write(decorated.encode()); err != nil {
		fmt.Fprintf(os.Stderr, "MSFS adapter UDP output failed: %v\n", err)
		return false
	}
	if session.firstPacketSent.IsZero() {
		session.firstPacketSent = time.Now()
	}
	session.lastForwarded = time.Now()
	return true
}

func (session *adapterSession) drainReplies() {
	buffer := make([]byte, 64)
	for {
		_ = session.connection.SetReadDeadline(time.Now().Add(time.Millisecond))
		n, err := session.connection.Read(buffer)
		if err != nil {
			return
		}
		if n >= packetSize && string(buffer[:4]) == "PXNF" && buffer[4] == version && !session.serviceAckSeen {
			session.serviceAckSeen = true
			fmt.Println("PXNF/1 service acknowledged MSFS adapter")
		}
	}
}

func (session *adapterSession) sendAction(action lifecycleAction) {
	if !action.Send {
		return
	}
	session.sequence++
	session.send(packet{Command: action.Command, Sequence: session.sequence}, action.StateFlags)
	if action.StateFlags&stateExiting != 0 {
		session.exitSent = true
	}
}

func (session *adapterSession) currentStateFlags() uint16 {
	return session.state.heartbeat().StateFlags
}

func (session *adapterSession) resetTelemetry() {
	session.mixer.reset()
	session.telemetryActive = false
}

func (session *adapterSession) handleTelemetry(data simTelemetry) {
	if !session.telemetryEnabled || session.state.Paused || !session.state.SimActive {
		return
	}
	now := time.Now()
	if now.Before(session.externalUntil) || now.Sub(session.lastTelemetrySent) < session.telemetryInterval {
		return
	}
	session.lastTelemetrySent = now
	stick, throttle := session.mixer.mix(data, session.telemetryGain)
	if stick == 0 && throttle == 0 {
		if session.telemetryActive {
			session.sequence++
			session.send(packet{Command: commandStop, Sequence: session.sequence}, session.currentStateFlags())
			session.telemetryActive = false
		}
		return
	}
	session.sequence++
	session.send(packet{
		Command: commandSet, Stick: stick, Throttle: throttle,
		Duration: uint16(telemetryCommandDuration / time.Millisecond),
		Sequence: session.sequence,
	}, session.currentStateFlags())
	session.telemetryActive = true
	if now.Sub(session.lastTelemetryLog) >= time.Second {
		session.lastTelemetryLog = now
		fmt.Printf("MSFS telemetry: stick=%d throttle=%d rpm=%.0f n1=%.1f ground=%.1f stall=%.0f\n",
			stick, throttle, data.EngineRPM, data.EngineN1, data.GroundVelocity, data.StallWarning)
	}
}

func (session *adapterSession) dispatch(data, size uintptr) {
	if data == 0 || size < unsafe.Sizeof(recvHeader{}) {
		return
	}
	header := (*recvHeader)(unsafe.Pointer(data))
	switch header.ID {
	case recvIDOpen:
		if size < unsafe.Sizeof(recvOpen{}) {
			return
		}
		message := (*recvOpen)(unsafe.Pointer(data))
		application := cString(message.ApplicationName[:])
		action := session.state.onOpen(application, message.ApplicationVersionMajor)
		session.simConnected = true
		fmt.Printf("SimConnect connected: %s version %d.%d; detected %s\n",
			application, message.ApplicationVersionMajor, message.ApplicationVersionMinor,
			adapterName(session.state.Adapter))
		session.sendAction(action)
	case recvIDEvent:
		if size < unsafe.Sizeof(recvEvent{}) {
			return
		}
		message := (*recvEvent)(unsafe.Pointer(data))
		switch message.EventID {
		case pauseEventID:
			paused := message.Data != 0
			if paused {
				session.resetTelemetry()
			}
			session.sendAction(session.state.onPause(paused))
			if paused {
				fmt.Println("MSFS Pause: stopping stick and throttle motors")
			} else {
				fmt.Println("MSFS resumed")
			}
		case simStartEventID:
			session.lastSimStart = time.Now()
			session.telemetrySeen = false
			session.telemetryWarned = false
			session.sendAction(session.state.onSimStart())
			fmt.Println("MSFS SimStart")
		case simStopEventID:
			session.resetTelemetry()
			session.sendAction(session.state.onSimStop())
			fmt.Println("MSFS SimStop: stopping stick and throttle motors")
		}
	case recvIDClientData:
		offset := unsafe.Offsetof(recvClientData{}.Data)
		if size < offset+packetSize {
			return
		}
		message := (*recvClientData)(unsafe.Pointer(data))
		if message.RequestID != requestID || message.DefineCount == 0 {
			return
		}
		raw := unsafe.Slice((*byte)(unsafe.Add(unsafe.Pointer(data), offset)), packetSize)
		if command, ok := decodePacket(raw); ok {
			if command.Command == commandSet || command.Command == commandStop {
				duration := time.Duration(command.Duration) * time.Millisecond
				if duration <= 0 {
					duration = 250 * time.Millisecond
				}
				session.externalUntil = time.Now().Add(duration)
			}
			session.send(command, session.currentStateFlags())
		}
	case recvIDSimObjectData:
		offset := unsafe.Offsetof(recvSimObjectData{}.Data)
		required := offset + uintptr(len(telemetryDefinitions))*unsafe.Sizeof(float64(0))
		if size < required {
			return
		}
		message := (*recvSimObjectData)(unsafe.Pointer(data))
		if message.RequestID != telemetryRequestID || message.DefineCount == 0 {
			return
		}
		if !session.telemetrySeen {
			session.telemetrySeen = true
			fmt.Printf("MSFS telemetry stream active: %d SimVars\n", len(telemetryDefinitions))
		}
		values := unsafe.Slice((*float64)(unsafe.Add(unsafe.Pointer(data), offset)), len(telemetryDefinitions))
		session.handleTelemetry(simTelemetry{
			AccelX: values[0], AccelY: values[1], AccelZ: values[2], GForce: values[3],
			Airspeed: values[4], VerticalSpeed: values[5], OnGround: values[6],
			StallWarning: values[7], GroundVelocity: values[8], EngineRPM: values[9],
			EngineN1: values[10], EngineCombustion: values[11],
		})
	case recvIDQuit:
		session.resetTelemetry()
		session.sendAction(session.state.onQuit())
		fmt.Println("MSFS Quit: stopping stick and throttle motors; adapter exiting")
		session.running = false
	case recvIDException:
		if size >= unsafe.Sizeof(recvHeader{})+4 {
			exception := *(*uint32)(unsafe.Add(unsafe.Pointer(data), unsafe.Sizeof(recvHeader{})))
			fmt.Fprintf(os.Stderr, "SimConnect exception: %d\n", exception)
		}
	}
}

func main() {
	serviceAddress := flag.String("service", "127.0.0.1:21190", "PXNF/1 UDP service endpoint")
	parentPID := flag.Uint("parent-pid", 0, "exit when this bridge process exits")
	connectTimeout := flag.Duration("connect-timeout", 5*time.Minute, "maximum time to wait for SimConnect")
	retryInterval := flag.Duration("retry", time.Second, "SimConnect retry interval")
	selfTest := flag.Bool("self-test", false, "validate version and lifecycle state handling without starting MSFS")
	telemetryEnabled := flag.Bool("telemetry", true, "generate simulator-owned vibration from MSFS SimVars")
	telemetryGain := flag.Float64("telemetry-gain", 1.25, "MSFS SimVar vibration gain before clamping")
	telemetryInterval := flag.Duration("telemetry-interval", 50*time.Millisecond, "MSFS SimVar vibration update interval")
	flag.Parse()
	if *selfTest {
		if err := runSelfTest(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	if *connectTimeout <= 0 || *retryInterval <= 0 {
		fmt.Fprintln(os.Stderr, "connect-timeout and retry must be positive")
		os.Exit(2)
	}
	if *telemetryGain < 0 || *telemetryGain > 4 || *telemetryInterval < 20*time.Millisecond || *telemetryInterval > time.Second {
		fmt.Fprintln(os.Stderr, "telemetry-gain must be 0..4 and telemetry-interval must be 20ms..1s")
		os.Exit(2)
	}
	address, err := net.ResolveUDPAddr("udp", *serviceAddress)
	if err != nil || !address.IP.IsLoopback() {
		fmt.Fprintf(os.Stderr, "service must be a loopback UDP address: %s\n", *serviceAddress)
		os.Exit(2)
	}
	connection, err := net.DialUDP("udp", nil, address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect PXNF/1 service: %v\n", err)
		os.Exit(2)
	}
	defer connection.Close()

	api, dllPath, err := loadSimConnect()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "Place the MSFS SDK SimConnect.dll beside PXNMSFSAdapter.exe if Windows cannot locate it.")
		os.Exit(3)
	}
	defer api.dll.Release()
	fmt.Printf("SimConnect runtime loaded: %s\n", dllPath)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	defer signal.Stop(interrupt)
	deadline := time.Now().Add(*connectTimeout)
	var handle syscall.Handle
	seenSimulatorProcess := false
	for {
		if uint32(*parentPID) != 0 {
			alive, checkErr := winproc.Alive(uint32(*parentPID))
			if checkErr == nil && !alive {
				fmt.Println("Bridge process exited before SimConnect became available")
				return
			}
		}
		if _, running, checkErr := winproc.FindMSFS(); checkErr == nil {
			if running {
				seenSimulatorProcess = true
			} else if seenSimulatorProcess {
				fmt.Println("MSFS process exited before SimConnect became available")
				return
			}
		}
		handle, err = api.connect()
		if err == nil {
			break
		}
		if time.Now().After(deadline) {
			fmt.Fprintf(os.Stderr, "SimConnect connection timed out: %v\n", err)
			os.Exit(4)
		}
		select {
		case <-interrupt:
			return
		case <-time.After(*retryInterval):
		}
	}
	defer api.close.Call(uintptr(handle))
	if err := api.setup(handle); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(5)
	}

	session := &adapterSession{
		api:               api,
		handle:            handle,
		connection:        connection,
		state:             newRuntimeState(),
		running:           true,
		telemetryEnabled:  *telemetryEnabled,
		telemetryGain:     *telemetryGain,
		telemetryInterval: *telemetryInterval,
	}
	if session.telemetryEnabled {
		fmt.Printf("MSFS SimVar vibration enabled: gain=%.2f interval=%s\n", session.telemetryGain, session.telemetryInterval)
	}
	activeSession = session
	defer func() { activeSession = nil }()
	defer func() {
		if !session.exitSent {
			session.sendAction(session.state.onQuit())
		}
	}()

	lastProcessCheck := time.Time{}
	missingProcessChecks := 0
	for session.running {
		select {
		case <-interrupt:
			fmt.Println("MSFS adapter interrupted")
			return
		default:
		}
		now := time.Now()
		if now.Sub(lastProcessCheck) >= processCheckInterval {
			lastProcessCheck = now
			if uint32(*parentPID) != 0 {
				alive, checkErr := winproc.Alive(uint32(*parentPID))
				if checkErr == nil && !alive {
					fmt.Println("Bridge process exited; adapter stopping")
					return
				}
			}
			_, running, checkErr := winproc.FindMSFS()
			if checkErr == nil && !running {
				missingProcessChecks++
				if missingProcessChecks >= 4 {
					fmt.Println("MSFS process exited without SimConnect Quit; adapter stopping")
					return
				}
			} else {
				missingProcessChecks = 0
			}
		}
		result, _, _ := api.callDispatch.Call(uintptr(handle), callbackPointer, 0)
		if err := checkHRESULT("SimConnect_CallDispatch", result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		if time.Since(session.lastForwarded) >= heartbeatInterval {
			session.sendAction(session.state.heartbeat())
		}
		session.drainReplies()
		if !session.serviceAckSeen && !session.serviceAckWarned && !session.firstPacketSent.IsZero() && time.Since(session.firstPacketSent) >= 2*time.Second {
			session.serviceAckWarned = true
			fmt.Fprintln(os.Stderr, "PXNF/1 service has not acknowledged adapter packets; check bridge version and UDP port 21190")
		}
		if session.telemetryEnabled && session.state.SimActive && !session.state.Paused && !session.telemetrySeen && !session.telemetryWarned && !session.lastSimStart.IsZero() && time.Since(session.lastSimStart) >= 5*time.Second {
			session.telemetryWarned = true
			fmt.Fprintln(os.Stderr, "MSFS telemetry stream not received after SimStart; check preceding SimConnect exceptions")
		}
		time.Sleep(10 * time.Millisecond)
	}
}
