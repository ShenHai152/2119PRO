package main

import (
	"encoding/binary"
	"testing"
	"time"
)

func makeServicePacket(command, left, right byte, duration uint16, sequence uint32) []byte {
	return makeServicePacketWithFlags(command, left, right, duration, 0, sequence)
}

func makeServicePacketWithFlags(command, left, right byte, duration, flags uint16, sequence uint32) []byte {
	packet := make([]byte, servicePacketSize)
	copy(packet[:4], "PXNF")
	packet[4] = serviceVersion
	packet[5] = command
	packet[6] = left
	packet[7] = right
	binary.LittleEndian.PutUint16(packet[8:10], duration)
	binary.LittleEndian.PutUint16(packet[10:12], flags)
	binary.LittleEndian.PutUint32(packet[12:16], sequence)
	return packet
}

func TestSimulatorIdentityAndStateFlags(t *testing.T) {
	packet, ok := decodeServicePacket(makeServicePacketWithFlags(
		serviceCommandStop, 0, 0, 0,
		serviceAdapterMSFS2024|serviceStatePaused, 10,
	))
	if !ok {
		t.Fatal("packet rejected")
	}
	if packet.Flags != serviceAdapterMSFS2024|serviceStatePaused {
		t.Fatalf("flags=%#x", packet.Flags)
	}
	if got := simulatorAdapterName(packet.Flags); got != "MSFS 2024" {
		t.Fatalf("adapter=%q", got)
	}
	if got := simulatorPacketState(packet); got != "paused" {
		t.Fatalf("state=%q", got)
	}
	packet.Flags = serviceAdapterXPlane12 | serviceStateExiting
	if got := simulatorAdapterName(packet.Flags); got != "X-Plane 12" {
		t.Fatalf("adapter=%q", got)
	}
	if got := simulatorPacketState(packet); got != "exiting" {
		t.Fatalf("state=%q", got)
	}
}

func TestServicePreservesIndependentChannels(t *testing.T) {
	packet, ok := decodeServicePacket(makeServicePacket(serviceCommandSet, 40, 190, 300, 7))
	if !ok {
		t.Fatal("packet rejected")
	}
	state := serviceState{}
	now := time.Unix(100, 0)
	command, changed := state.accept(packet, now)
	if !changed || command.Left != 40 || command.Right != 190 {
		t.Fatalf("command=%#v changed=%t", command, changed)
	}
	if !state.deadline.Equal(now.Add(300 * time.Millisecond)) {
		t.Fatalf("deadline=%v", state.deadline)
	}
}

func TestServiceDefaultAndMaximumWatchdog(t *testing.T) {
	defaultPacket, ok := decodeServicePacket(makeServicePacket(serviceCommandSet, 1, 2, 0, 1))
	if !ok || defaultPacket.Duration != serviceDefaultTimeout {
		t.Fatalf("default duration=%v ok=%t", defaultPacket.Duration, ok)
	}
	maxPacket, ok := decodeServicePacket(makeServicePacket(serviceCommandSet, 1, 2, 65000, 2))
	if !ok || maxPacket.Duration != serviceMaxTimeout {
		t.Fatalf("max duration=%v ok=%t", maxPacket.Duration, ok)
	}
}

func TestServiceWatchdogStopsBothChannels(t *testing.T) {
	state := serviceState{}
	now := time.Unix(200, 0)
	packet, _ := decodeServicePacket(makeServicePacket(serviceCommandSet, 20, 220, 100, 9))
	state.accept(packet, now)
	if _, changed := state.expire(now.Add(99 * time.Millisecond)); changed {
		t.Fatal("watchdog expired too early")
	}
	command, changed := state.expire(now.Add(100 * time.Millisecond))
	if !changed || command.Left != 0 || command.Right != 0 {
		t.Fatalf("expiry command=%#v changed=%t", command, changed)
	}
}

func TestServiceRejectsInvalidPacket(t *testing.T) {
	if _, ok := decodeServicePacket([]byte("short")); ok {
		t.Fatal("accepted short packet")
	}
	packet := makeServicePacket(serviceCommandSet, 1, 2, 100, 1)
	packet[4] = 2
	if _, ok := decodeServicePacket(packet); ok {
		t.Fatal("accepted unknown version")
	}
}
