package main

import "testing"

func TestDetectAdapterFromOpen(t *testing.T) {
	tests := []struct {
		name  string
		major uint32
		want  uint16
	}{
		{"Microsoft Flight Simulator", 11, adapterMSFS2020},
		{"KittyHawk", 11, adapterMSFS2020},
		{"Microsoft Flight Simulator 2024", 11, adapterMSFS2024},
		{"FlightSimulator", 12, adapterMSFS2024},
		{"Unknown Sim", 1, adapterMSFSGeneric},
	}
	for _, test := range tests {
		if got := detectAdapter(test.name, test.major); got != test.want {
			t.Fatalf("detectAdapter(%q,%d)=%#x want %#x", test.name, test.major, got, test.want)
		}
	}
}

func TestPauseStopsBothChannelsAndBlocksSet(t *testing.T) {
	state := newRuntimeState()
	state.onOpen("Microsoft Flight Simulator", 11)
	action := state.onPause(true)
	if !action.Send || action.Command != commandStop || action.StateFlags != statePaused {
		t.Fatalf("pause action=%#v", action)
	}
	if _, ok := state.decorate(packet{Command: commandSet, Stick: 200, Throttle: 180}, 0); ok {
		t.Fatal("SET forwarded while paused")
	}
	resume := state.onPause(false)
	if resume.Command != commandPing || !resume.Send {
		t.Fatalf("resume action=%#v", resume)
	}
}

func TestSimStopAndQuitSendImmediateStop(t *testing.T) {
	state := newRuntimeState()
	state.Paused = true
	start := state.onSimStart()
	if state.Paused || !state.SimActive || start.Command != commandPing {
		t.Fatalf("SimStart did not resume state: state=%#v action=%#v", state, start)
	}
	stop := state.onSimStop()
	if stop.Command != commandStop || stop.StateFlags != stateStopped || stop.Exit {
		t.Fatalf("SimStop action=%#v", stop)
	}
	quit := state.onQuit()
	if quit.Command != commandStop || quit.StateFlags != stateExiting || !quit.Exit {
		t.Fatalf("Quit action=%#v", quit)
	}
}

func TestPacketRoundTripAndAdapterDecoration(t *testing.T) {
	state := newRuntimeState()
	state.onOpen("Microsoft Flight Simulator 2024", 12)
	input := packet{Command: commandSet, Stick: 33, Throttle: 211, Duration: 250, Sequence: 77}
	decorated, ok := state.decorate(input, 0)
	if !ok || decorated.Flags&adapterMask != adapterMSFS2024 {
		t.Fatalf("decorated=%#v ok=%t", decorated, ok)
	}
	decoded, ok := decodePacket(decorated.encode())
	if !ok || decoded != decorated {
		t.Fatalf("decoded=%#v decorated=%#v ok=%t", decoded, decorated, ok)
	}
}
