package main

import "math"

type simTelemetry struct {
	AccelX           float64
	AccelY           float64
	AccelZ           float64
	GForce           float64
	Airspeed         float64
	VerticalSpeed    float64
	OnGround         float64
	StallWarning     float64
	GroundVelocity   float64
	EngineRPM        float64
	EngineN1         float64
	EngineCombustion float64
}

type telemetryMixer struct {
	initialized bool
	lastAccelX  float64
	lastAccelY  float64
	lastAccelZ  float64
	stick       float64
	throttle    float64
}

func clampFloat(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func maxFloat(values ...float64) float64 {
	maximum := 0.0
	for _, value := range values {
		if value > maximum {
			maximum = value
		}
	}
	return maximum
}

func smoothAmplitude(current, target float64) float64 {
	alpha := 0.12
	if target > current {
		alpha = 0.55
	}
	value := current + (target-current)*alpha
	if value < 0.008 && target == 0 {
		return 0
	}
	return value
}

func amplitudeByte(value float64) byte {
	return byte(math.Round(clampFloat(value, 0, 1) * 255))
}

// mix maps simulator-owned state to the two physical eccentric motors. The
// joystick-base channel represents airframe/ground/stall load. The throttle
// channel represents engine and powerplant vibration. External PXNF ClientData
// commands remain able to temporarily override these generated amplitudes.
func (m *telemetryMixer) mix(data simTelemetry, gain float64) (byte, byte) {
	jerk := 0.0
	if m.initialized {
		delta := math.Abs(data.AccelX-m.lastAccelX) +
			math.Abs(data.AccelY-m.lastAccelY) +
			math.Abs(data.AccelZ-m.lastAccelZ)
		jerk = clampFloat(delta/(32.174*0.8), 0, 1)
	}
	m.initialized = true
	m.lastAccelX = data.AccelX
	m.lastAccelY = data.AccelY
	m.lastAccelZ = data.AccelZ

	gLoad := clampFloat((math.Abs(data.GForce-1)-0.04)/0.9, 0, 1)
	ground := 0.0
	if data.OnGround >= 0.5 {
		ground = clampFloat(data.GroundVelocity/70, 0, 1)
	}
	stall := 0.0
	if data.StallWarning >= 0.5 {
		stall = 1
	}
	engineLevel := maxFloat(
		clampFloat(data.EngineRPM/2700, 0, 1),
		clampFloat(data.EngineN1/100, 0, 1),
	)
	engine := 0.0
	if data.EngineCombustion >= 0.5 || engineLevel > 0.03 {
		engine = 0.18 + engineLevel*0.38
	}

	stickTarget := maxFloat(
		stall*0.82,
		ground*0.48,
		gLoad*0.58+jerk*0.62,
		engine*0.24,
	)
	throttleTarget := maxFloat(
		engine,
		ground*0.16+engine*0.88,
		jerk*0.12+engine,
	)
	stickTarget = clampFloat(stickTarget*gain, 0, 1)
	throttleTarget = clampFloat(throttleTarget*gain, 0, 1)

	m.stick = smoothAmplitude(m.stick, stickTarget)
	m.throttle = smoothAmplitude(m.throttle, throttleTarget)
	return amplitudeByte(m.stick), amplitudeByte(m.throttle)
}

func (m *telemetryMixer) reset() {
	*m = telemetryMixer{}
}
