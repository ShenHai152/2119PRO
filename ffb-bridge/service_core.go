package main

import (
	"encoding/binary"
	"time"
)

const (
	servicePacketSize     = 16
	serviceVersion        = 1
	serviceCommandSet     = 1
	serviceCommandStop    = 2
	serviceCommandPing    = 3
	serviceDefaultTimeout = 250 * time.Millisecond
	serviceMaxTimeout     = 5 * time.Second
	serviceAdapterTimeout = 1500 * time.Millisecond

	serviceAdapterMask        = 0x000f
	serviceAdapterUnknown     = 0x0000
	serviceAdapterMSFS2020    = 0x0001
	serviceAdapterMSFS2024    = 0x0002
	serviceAdapterXPlane12    = 0x0003
	serviceAdapterMSFSGeneric = 0x0004

	serviceStatePaused  = 0x0100
	serviceStateStopped = 0x0200
	serviceStateExiting = 0x0400
)

type motorCommand struct {
	Left  byte
	Right byte
}

type servicePacket struct {
	Command  byte
	Left     byte
	Right    byte
	Duration time.Duration
	Flags    uint16
	Sequence uint32
}

func decodeServicePacket(data []byte) (servicePacket, bool) {
	if len(data) < servicePacketSize || string(data[:4]) != "PXNF" || data[4] != serviceVersion {
		return servicePacket{}, false
	}
	duration := time.Duration(binary.LittleEndian.Uint16(data[8:10])) * time.Millisecond
	if duration <= 0 {
		duration = serviceDefaultTimeout
	}
	if duration > serviceMaxTimeout {
		duration = serviceMaxTimeout
	}
	return servicePacket{
		Command:  data[5],
		Left:     data[6],
		Right:    data[7],
		Duration: duration,
		Flags:    binary.LittleEndian.Uint16(data[10:12]),
		Sequence: binary.LittleEndian.Uint32(data[12:16]),
	}, true
}

func simulatorAdapterName(flags uint16) string {
	switch flags & serviceAdapterMask {
	case serviceAdapterMSFS2020:
		return "MSFS 2020"
	case serviceAdapterMSFS2024:
		return "MSFS 2024"
	case serviceAdapterXPlane12:
		return "X-Plane 12"
	case serviceAdapterMSFSGeneric:
		return "Microsoft Flight Simulator"
	default:
		return "Legacy/unknown PXNF/1"
	}
}

func simulatorPacketState(packet servicePacket) string {
	switch {
	case packet.Flags&serviceStateExiting != 0:
		return "exiting"
	case packet.Flags&serviceStatePaused != 0:
		return "paused"
	case packet.Flags&serviceStateStopped != 0:
		return "stopped"
	case packet.Command == serviceCommandSet:
		return "active"
	default:
		return "idle"
	}
}

type serviceState struct {
	active   bool
	deadline time.Time
	lastSeq  uint32
}

func (s *serviceState) accept(packet servicePacket, now time.Time) (motorCommand, bool) {
	s.lastSeq = packet.Sequence
	switch packet.Command {
	case serviceCommandSet:
		s.active = true
		s.deadline = now.Add(packet.Duration)
		return motorCommand{Left: packet.Left, Right: packet.Right}, true
	case serviceCommandStop:
		s.active = false
		return motorCommand{}, true
	default:
		return motorCommand{}, false
	}
}

func (s *serviceState) expire(now time.Time) (motorCommand, bool) {
	if s.active && !now.Before(s.deadline) {
		s.active = false
		return motorCommand{}, true
	}
	return motorCommand{}, false
}
