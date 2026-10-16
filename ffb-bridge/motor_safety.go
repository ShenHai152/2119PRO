package main

import "fmt"

const (
	defaultStickMaximum    = 220
	defaultThrottleMaximum = 220
	defaultMotorBudget     = 330
	defaultMotorSlew       = 24
)

// motorSafetyConfig limits the final calibrated bytes immediately before they
// reach the HID backend. This protects every simulator adapter from requesting
// a sustained dual-motor load that can brown out or trip protection in the
// controller. MotorSlew limits rising edges per output-loop update; falling
// edges, including STOP, are immediate.
type motorSafetyConfig struct {
	StickMaximum    int
	ThrottleMaximum int
	TotalBudget     int
	MotorSlew       int
}

func defaultMotorSafetyConfig() motorSafetyConfig {
	return motorSafetyConfig{
		StickMaximum:    defaultStickMaximum,
		ThrottleMaximum: defaultThrottleMaximum,
		TotalBudget:     defaultMotorBudget,
		MotorSlew:       defaultMotorSlew,
	}
}

func (c motorSafetyConfig) validate() error {
	if c.StickMaximum < 0 || c.StickMaximum > 255 {
		return fmt.Errorf("stick maximum must be within 0..255")
	}
	if c.ThrottleMaximum < 0 || c.ThrottleMaximum > 255 {
		return fmt.Errorf("throttle maximum must be within 0..255")
	}
	if c.TotalBudget < 1 || c.TotalBudget > 510 {
		return fmt.Errorf("motor budget must be within 1..510")
	}
	if c.MotorSlew < 1 || c.MotorSlew > 255 {
		return fmt.Errorf("motor slew must be within 1..255")
	}
	return nil
}

type motorSafetyLimiter struct {
	config  motorSafetyConfig
	target  motorCommand
	current motorCommand
}

func newMotorSafetyLimiter(config motorSafetyConfig) *motorSafetyLimiter {
	return &motorSafetyLimiter{config: config}
}

func minInt(value, maximum int) int {
	if value > maximum {
		return maximum
	}
	return value
}

// constrain applies per-channel electrical limits followed by a proportional
// shared budget. Proportional scaling preserves the simulator's requested
// stick/throttle balance instead of arbitrarily starving one motor.
func (l *motorSafetyLimiter) constrain(requested motorCommand) (motorCommand, bool) {
	stick := minInt(int(requested.Left), l.config.StickMaximum)
	throttle := minInt(int(requested.Right), l.config.ThrottleMaximum)
	limited := stick != int(requested.Left) || throttle != int(requested.Right)
	total := stick + throttle
	if total > l.config.TotalBudget {
		originalStick := stick
		originalThrottle := throttle
		stick = originalStick * l.config.TotalBudget / total
		throttle = originalThrottle * l.config.TotalBudget / total
		// Preserve a real low-amplitude channel when the shared budget has room
		// for both active motors. Without this, integer scaling could turn a
		// requested 1 into 0 and change the simulator's channel semantics.
		if l.config.TotalBudget >= 2 {
			if originalStick > 0 && stick == 0 {
				stick = 1
			}
			if originalThrottle > 0 && throttle == 0 {
				throttle = 1
			}
		}
		for stick+throttle > l.config.TotalBudget {
			if throttle >= stick && throttle > 1 {
				throttle--
			} else if stick > 1 {
				stick--
			} else {
				break
			}
		}
		// Integer division may leave one budget unit unused. Assign it to the
		// channel with the larger discarded remainder without exceeding its cap.
		remaining := l.config.TotalBudget - stick - throttle
		if remaining > 0 {
			stickRemainder := originalStick*l.config.TotalBudget - stick*total
			throttleRemainder := originalThrottle*l.config.TotalBudget - throttle*total
			if stickRemainder >= throttleRemainder && stick < originalStick {
				stick++
			} else if throttle < originalThrottle {
				throttle++
			}
		}
		limited = true
	}
	return motorCommand{Left: byte(stick), Right: byte(throttle)}, limited
}

func (l *motorSafetyLimiter) setTarget(requested motorCommand) (motorCommand, bool) {
	target, limited := l.constrain(requested)
	l.target = target
	return target, limited
}

func riseToward(current, target byte, maximumStep int) byte {
	if target <= current {
		return target
	}
	delta := int(target) - int(current)
	if delta <= maximumStep {
		return target
	}
	return byte(int(current) + maximumStep)
}

// advance applies a bounded rising edge. Reductions are immediate so Pause,
// SimStop, Quit, watchdog expiry, and adapter disconnect always reach zero in
// the same output-loop iteration.
func (l *motorSafetyLimiter) advance() (motorCommand, bool) {
	next := motorCommand{
		Left:  riseToward(l.current.Left, l.target.Left, l.config.MotorSlew),
		Right: riseToward(l.current.Right, l.target.Right, l.config.MotorSlew),
	}
	changed := next != l.current
	l.current = next
	return next, changed
}

func (l *motorSafetyLimiter) reset() {
	l.target = motorCommand{}
	l.current = motorCommand{}
}
