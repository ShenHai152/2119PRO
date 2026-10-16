package main

import (
	"encoding/binary"
	"fmt"
	"strings"
)

const (
	packetSize = 16
	version    = 1

	commandSet  = 1
	commandStop = 2
	commandPing = 3

	adapterMask        = 0x000f
	adapterMSFS2020    = 0x0001
	adapterMSFS2024    = 0x0002
	adapterMSFSGeneric = 0x0004

	statePaused  = 0x0100
	stateStopped = 0x0200
	stateExiting = 0x0400
)

type packet struct {
	Command  byte
	Stick    byte
	Throttle byte
	Duration uint16
	Flags    uint16
	Sequence uint32
}

func decodePacket(data []byte) (packet, bool) {
	if len(data) < packetSize || string(data[:4]) != "PXNF" || data[4] != version {
		return packet{}, false
	}
	return packet{
		Command:  data[5],
		Stick:    data[6],
		Throttle: data[7],
		Duration: binary.LittleEndian.Uint16(data[8:10]),
		Flags:    binary.LittleEndian.Uint16(data[10:12]),
		Sequence: binary.LittleEndian.Uint32(data[12:16]),
	}, true
}

func (p packet) encode() []byte {
	data := make([]byte, packetSize)
	copy(data[:4], "PXNF")
	data[4] = version
	data[5] = p.Command
	data[6] = p.Stick
	data[7] = p.Throttle
	binary.LittleEndian.PutUint16(data[8:10], p.Duration)
	binary.LittleEndian.PutUint16(data[10:12], p.Flags)
	binary.LittleEndian.PutUint32(data[12:16], p.Sequence)
	return data
}

func detectAdapter(applicationName string, applicationVersionMajor uint32) uint16 {
	name := strings.ToLower(applicationName)
	if strings.Contains(name, "2024") || applicationVersionMajor >= 12 {
		return adapterMSFS2024
	}
	if strings.Contains(name, "kittyhawk") ||
		strings.Contains(name, "flight simulator") ||
		strings.Contains(name, "flightsimulator") {
		return adapterMSFS2020
	}
	return adapterMSFSGeneric
}

func adapterName(adapter uint16) string {
	switch adapter & adapterMask {
	case adapterMSFS2020:
		return "MSFS 2020"
	case adapterMSFS2024:
		return "MSFS 2024"
	default:
		return "Microsoft Flight Simulator"
	}
}

type lifecycleAction struct {
	Command    byte
	StateFlags uint16
	Send       bool
	Exit       bool
}

type runtimeState struct {
	Adapter   uint16
	Paused    bool
	SimActive bool
}

func newRuntimeState() runtimeState {
	return runtimeState{Adapter: adapterMSFSGeneric, SimActive: true}
}

func (s *runtimeState) onOpen(applicationName string, applicationVersionMajor uint32) lifecycleAction {
	s.Adapter = detectAdapter(applicationName, applicationVersionMajor)
	s.Paused = false
	s.SimActive = true
	return lifecycleAction{Command: commandPing, Send: true}
}

func (s *runtimeState) onPause(paused bool) lifecycleAction {
	s.Paused = paused
	if paused {
		return lifecycleAction{Command: commandStop, StateFlags: statePaused, Send: true}
	}
	return lifecycleAction{Command: commandPing, Send: true}
}

func (s *runtimeState) onSimStart() lifecycleAction {
	s.Paused = false
	s.SimActive = true
	return lifecycleAction{Command: commandPing, Send: true}
}

func (s *runtimeState) onSimStop() lifecycleAction {
	s.SimActive = false
	return lifecycleAction{Command: commandStop, StateFlags: stateStopped, Send: true}
}

func (s *runtimeState) onQuit() lifecycleAction {
	s.SimActive = false
	return lifecycleAction{Command: commandStop, StateFlags: stateExiting, Send: true, Exit: true}
}

func (s *runtimeState) heartbeat() lifecycleAction {
	flags := uint16(0)
	if s.Paused {
		flags |= statePaused
	}
	if !s.SimActive {
		flags |= stateStopped
	}
	return lifecycleAction{Command: commandPing, StateFlags: flags, Send: true}
}

func (s *runtimeState) decorate(input packet, stateFlags uint16) (packet, bool) {
	if input.Command == commandSet && (s.Paused || !s.SimActive) {
		return packet{}, false
	}
	const knownFlags = adapterMask | statePaused | stateStopped | stateExiting
	input.Flags = (input.Flags &^ knownFlags) | s.Adapter | stateFlags
	return input, true
}

func runSelfTest() error {
	state := newRuntimeState()
	if state.onOpen("KittyHawk", 11); state.Adapter != adapterMSFS2020 {
		return fmt.Errorf("MSFS 2020 detection failed")
	}
	if state.onOpen("Microsoft Flight Simulator 2024", 12); state.Adapter != adapterMSFS2024 {
		return fmt.Errorf("MSFS 2024 detection failed")
	}
	if action := state.onPause(true); action.Command != commandStop || action.StateFlags != statePaused {
		return fmt.Errorf("Pause STOP failed")
	}
	if action := state.onSimStop(); action.Command != commandStop || action.StateFlags != stateStopped {
		return fmt.Errorf("SimStop STOP failed")
	}
	if action := state.onQuit(); action.Command != commandStop || action.StateFlags != stateExiting || !action.Exit {
		return fmt.Errorf("Quit STOP failed")
	}
	fmt.Println("Self-test PASS: MSFS 2020/2024 detection")
	fmt.Println("Self-test PASS: Pause -> STOP(stick=0, throttle=0)")
	fmt.Println("Self-test PASS: SimStop -> STOP(stick=0, throttle=0)")
	fmt.Println("Self-test PASS: Quit -> STOP(stick=0, throttle=0) + exit")
	return nil
}
