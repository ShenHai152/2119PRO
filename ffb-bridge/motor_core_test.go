package main

import (
	"errors"
	"testing"
	"time"
)

type fakeMotor struct {
	calls  [][2]float64
	fail   bool
	closed bool
}

func (m *fakeMotor) set(left, right float64) error {
	m.calls = append(m.calls, [2]float64{left, right})
	if m.fail {
		return errors.New("write failed")
	}
	return nil
}

func (m *fakeMotor) healthy() error {
	if m.fail {
		return errors.New("health failed")
	}
	return nil
}

func (m *fakeMotor) close() { m.closed = true }

func TestReconnectStartsAtZeroAndDoesNotReplay(t *testing.T) {
	first := &fakeMotor{}
	second := &fakeMotor{}
	opened := 0
	manager := newReconnectingMotor(func() (motorOutput, string, error) {
		opened++
		if opened == 1 {
			return first, "first", nil
		}
		return second, "second", nil
	}, time.Second)
	now := time.Unix(1000, 0)
	connected, _, err := manager.tick(now)
	if err != nil || !connected || len(first.calls) != 1 || first.calls[0] != [2]float64{0, 0} {
		t.Fatalf("initial connect: connected=%t err=%v calls=%v", connected, err, first.calls)
	}
	if err := manager.apply(now, 0.2, 0.8); err != nil {
		t.Fatal(err)
	}
	if first.calls[len(first.calls)-1] != [2]float64{0.2, 0.8} {
		t.Fatalf("independent channels lost: %v", first.calls)
	}
	first.fail = true
	if err := manager.apply(now, 0.7, 0.1); err == nil {
		t.Fatal("expected write failure")
	}
	if !first.closed || manager.connected() {
		t.Fatal("failed motor was not disconnected")
	}
	if connected, _, _ := manager.tick(now.Add(500 * time.Millisecond)); connected {
		t.Fatal("reconnected before retry interval")
	}
	connected, _, err = manager.tick(now.Add(time.Second))
	if err != nil || !connected {
		t.Fatalf("reconnect: connected=%t err=%v", connected, err)
	}
	if len(second.calls) != 1 || second.calls[0] != [2]float64{0, 0} {
		t.Fatalf("reconnect replayed output: %v", second.calls)
	}
}

func TestDisconnectedCommandIsRejected(t *testing.T) {
	manager := newReconnectingMotor(nil, time.Second)
	if err := manager.apply(time.Now(), 1, 1); !errors.Is(err, errMotorDisconnected) {
		t.Fatalf("got %v", err)
	}
}

func TestHealthFailureSchedulesReconnect(t *testing.T) {
	motor := &fakeMotor{}
	manager := newReconnectingMotor(func() (motorOutput, string, error) {
		return motor, "motor", nil
	}, time.Second)
	now := time.Unix(2000, 0)
	if connected, _, err := manager.tick(now); err != nil || !connected {
		t.Fatalf("connect: connected=%t err=%v", connected, err)
	}
	motor.fail = true
	if err := manager.healthCheck(now.Add(time.Second)); err == nil {
		t.Fatal("expected health failure")
	}
	if manager.connected() || !motor.closed {
		t.Fatal("unhealthy motor was not disconnected")
	}
	if connected, _, _ := manager.tick(now.Add(1500 * time.Millisecond)); connected {
		t.Fatal("health failure reconnected before retry interval")
	}
}
