package main

import "testing"

func TestTelemetryEngineDrivesThrottleMotor(t *testing.T) {
	mixer := telemetryMixer{}
	data := simTelemetry{
		GForce:           1,
		EngineRPM:        2400,
		EngineCombustion: 1,
	}
	var stick, throttle byte
	for index := 0; index < 4; index++ {
		stick, throttle = mixer.mix(data, 1)
	}
	if throttle < 90 {
		t.Fatalf("throttle amplitude too low: %d", throttle)
	}
	if stick >= throttle {
		t.Fatalf("engine should favor throttle motor: stick=%d throttle=%d", stick, throttle)
	}
}

func TestTelemetryGroundRollDrivesStickMotor(t *testing.T) {
	mixer := telemetryMixer{}
	stick, _ := mixer.mix(simTelemetry{GForce: 1, OnGround: 1, GroundVelocity: 70}, 1)
	if stick < 60 {
		t.Fatalf("ground-roll stick amplitude too low: %d", stick)
	}
}

func TestTelemetryStallIsStrongAndClamped(t *testing.T) {
	mixer := telemetryMixer{}
	stick, throttle := mixer.mix(simTelemetry{GForce: 1, StallWarning: 1}, 4)
	if stick < 110 {
		t.Fatalf("stall amplitude too low: %d", stick)
	}
	if throttle != 0 {
		t.Fatalf("stall without engine should not drive throttle: %d", throttle)
	}
}

func TestTelemetryResetClearsSmoothingState(t *testing.T) {
	mixer := telemetryMixer{}
	mixer.mix(simTelemetry{GForce: 2, EngineN1: 100}, 1)
	mixer.reset()
	stick, throttle := mixer.mix(simTelemetry{GForce: 1}, 1)
	if stick != 0 || throttle != 0 {
		t.Fatalf("reset output stick=%d throttle=%d", stick, throttle)
	}
}
