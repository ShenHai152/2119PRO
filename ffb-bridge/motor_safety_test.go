package main

import "testing"

func TestMotorSafetyPerChannelMaximums(t *testing.T) {
	limiter := newMotorSafetyLimiter(motorSafetyConfig{
		StickMaximum: 200, ThrottleMaximum: 180, TotalBudget: 510, MotorSlew: 255,
	})
	target, limited := limiter.setTarget(motorCommand{Left: 240, Right: 220})
	if !limited || target != (motorCommand{Left: 200, Right: 180}) {
		t.Fatalf("target=%#v limited=%t", target, limited)
	}
}

func TestMotorSafetySharedBudgetPreservesBalance(t *testing.T) {
	limiter := newMotorSafetyLimiter(motorSafetyConfig{
		StickMaximum: 255, ThrottleMaximum: 255, TotalBudget: 300, MotorSlew: 255,
	})
	target, limited := limiter.setTarget(motorCommand{Left: 240, Right: 160})
	if !limited || target.Left != 180 || target.Right != 120 {
		t.Fatalf("target=%#v limited=%t", target, limited)
	}
	if int(target.Left)+int(target.Right) > 300 {
		t.Fatalf("budget exceeded: %#v", target)
	}
}

func TestMotorSafetyRampsOnlyRisingEdges(t *testing.T) {
	limiter := newMotorSafetyLimiter(motorSafetyConfig{
		StickMaximum: 255, ThrottleMaximum: 255, TotalBudget: 510, MotorSlew: 24,
	})
	limiter.setTarget(motorCommand{Left: 100, Right: 70})
	first, changed := limiter.advance()
	if !changed || first != (motorCommand{Left: 24, Right: 24}) {
		t.Fatalf("first=%#v changed=%t", first, changed)
	}
	second, _ := limiter.advance()
	if second != (motorCommand{Left: 48, Right: 48}) {
		t.Fatalf("second=%#v", second)
	}
	limiter.setTarget(motorCommand{Left: 10, Right: 0})
	falling, _ := limiter.advance()
	if falling != (motorCommand{Left: 10, Right: 0}) {
		t.Fatalf("falling edge was not immediate: %#v", falling)
	}
}

func TestMotorSafetyStopIsImmediate(t *testing.T) {
	limiter := newMotorSafetyLimiter(defaultMotorSafetyConfig())
	limiter.setTarget(motorCommand{Left: 220, Right: 220})
	for index := 0; index < 4; index++ {
		limiter.advance()
	}
	limiter.setTarget(motorCommand{})
	stopped, changed := limiter.advance()
	if !changed || stopped != (motorCommand{}) {
		t.Fatalf("stop=%#v changed=%t", stopped, changed)
	}
}

func TestMotorSafetyEventuallyReachesTarget(t *testing.T) {
	limiter := newMotorSafetyLimiter(defaultMotorSafetyConfig())
	target, _ := limiter.setTarget(motorCommand{Left: 160, Right: 80})
	var got motorCommand
	for index := 0; index < 20; index++ {
		got, _ = limiter.advance()
	}
	if got != target {
		t.Fatalf("got=%#v target=%#v", got, target)
	}
}

func TestMotorSafetyDoesNotDropLowNonzeroChannel(t *testing.T) {
	limiter := newMotorSafetyLimiter(motorSafetyConfig{
		StickMaximum: 255, ThrottleMaximum: 255, TotalBudget: 100, MotorSlew: 255,
	})
	target, _ := limiter.setTarget(motorCommand{Left: 1, Right: 255})
	if target.Left == 0 {
		t.Fatalf("low nonzero stick channel was dropped: %#v", target)
	}
	if int(target.Left)+int(target.Right) > 100 {
		t.Fatalf("budget exceeded: %#v", target)
	}
}
