//go:build windows

package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type motorService struct {
	conn             *net.UDPConn
	state            serviceState
	lastSource       string
	lastFailure      string
	adapterName      string
	simulatorState   string
	adapterLastSeen  time.Time
	adapterConnected bool
}

type simulatorSnapshot struct {
	Adapter   string
	State     string
	Connected bool
}

func openMotorService(address string) (*motorService, error) {
	addr, err := net.ResolveUDPAddr("udp", address)
	if err != nil {
		return nil, fmt.Errorf("resolve service address %q: %w", address, err)
	}
	if !addr.IP.IsLoopback() {
		return nil, fmt.Errorf("motor service must listen on a loopback address, got %q", address)
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen for simulator vibration commands: %w", err)
	}
	return &motorService{conn: conn}, nil
}

func (s *motorService) close() {
	if s != nil && s.conn != nil {
		_ = s.conn.Close()
		s.conn = nil
	}
}

func (s *motorService) reply(addr *net.UDPAddr, command byte, left, right byte, flags uint16, sequence uint32) {
	response := make([]byte, servicePacketSize)
	copy(response[:4], "PXNF")
	response[4] = serviceVersion
	response[5] = command | 0x80
	response[6] = left
	response[7] = right
	binary.LittleEndian.PutUint16(response[10:12], flags)
	binary.LittleEndian.PutUint32(response[12:16], sequence)
	_, _ = s.conn.WriteToUDP(response, addr)
}

// poll consumes all currently queued localhost commands. It returns a new
// motor command only when a SET/STOP command or watchdog expiry changes the
// output. The service preserves independent left/right amplitudes and does not
// invent vibration content.
func (s *motorService) poll(now time.Time) (motorCommand, bool) {
	if s == nil || s.conn == nil {
		return motorCommand{}, false
	}
	buf := make([]byte, 64)
	var output motorCommand
	var changed bool
	for {
		// An already-expired deadline can return WSAETIMEDOUT before Windows
		// checks a queued UDP datagram. A short future deadline preserves the
		// non-blocking service loop while reliably consuming localhost packets.
		_ = s.conn.SetReadDeadline(time.Now().Add(time.Millisecond))
		n, addr, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
				break
			}
			message := err.Error()
			if message != s.lastFailure {
				fmt.Printf("PXN motor service receive error: %v\n", err)
				s.lastFailure = message
			}
			break
		}
		packet, valid := decodeServicePacket(buf[:n])
		if !valid {
			continue
		}
		if addr.String() != s.lastSource {
			fmt.Printf("Simulator vibration client connected: %s\n", addr.String())
			s.lastSource = addr.String()
		}
		adapterName := simulatorAdapterName(packet.Flags)
		if adapterName != s.adapterName {
			fmt.Printf("Simulator adapter identified: %s\n", adapterName)
			s.adapterName = adapterName
		}
		s.adapterConnected = true
		s.adapterLastSeen = now
		packetState := simulatorPacketState(packet)
		if packetState != s.simulatorState {
			s.simulatorState = packetState
			if packetState == "paused" || packetState == "stopped" || packetState == "exiting" {
				fmt.Printf("Simulator state: %s (%s); stopping motors\n", packetState, adapterName)
			}
		}
		switch packet.Command {
		case serviceCommandSet:
			s.reply(addr, packet.Command, packet.Left, packet.Right, packet.Flags, packet.Sequence)
			output, changed = s.state.accept(packet, now)
		case serviceCommandStop:
			s.reply(addr, packet.Command, 0, 0, packet.Flags, packet.Sequence)
			output, changed = s.state.accept(packet, now)
		case serviceCommandPing:
			s.reply(addr, packet.Command, packet.Left, packet.Right, packet.Flags, packet.Sequence)
		}
	}
	if expiredOutput, expired := s.state.expire(now); expired {
		s.simulatorState = "watchdog-stopped"
		fmt.Printf("Simulator vibration watchdog stopped output after sequence %d\n", s.state.lastSeq)
		return expiredOutput, true
	}
	if s.adapterConnected && now.Sub(s.adapterLastSeen) > serviceAdapterTimeout {
		wasActive := s.state.active
		s.state.active = false
		s.adapterConnected = false
		s.simulatorState = "disconnected"
		fmt.Printf("Simulator adapter disconnected: %s; stopping motors\n", s.adapterName)
		if wasActive || changed {
			return motorCommand{}, true
		}
	}
	return output, changed
}

func (s *motorService) snapshot() simulatorSnapshot {
	if s == nil {
		return simulatorSnapshot{Adapter: "none", State: "disconnected"}
	}
	adapter := s.adapterName
	if adapter == "" {
		adapter = "none"
	}
	state := s.simulatorState
	if state == "" {
		state = "waiting"
	}
	return simulatorSnapshot{Adapter: adapter, State: state, Connected: s.adapterConnected}
}
