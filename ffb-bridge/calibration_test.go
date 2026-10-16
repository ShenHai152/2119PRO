package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestChannelCalibrationApply(t *testing.T) {
	tests := []struct {
		name   string
		config channelCalibration
		input  byte
		want   byte
	}{
		{name: "zero stays zero", config: channelCalibration{Gain: 2, MinimumOutput: 80}, input: 0, want: 0},
		{name: "linear", config: channelCalibration{Gain: 1}, input: 120, want: 120},
		{name: "minimum output", config: channelCalibration{Gain: 1, MinimumOutput: 35}, input: 5, want: 35},
		{name: "gain", config: channelCalibration{Gain: 1.5}, input: 100, want: 150},
		{name: "clamp", config: channelCalibration{Gain: 4}, input: 100, want: 255},
		{name: "disabled channel", config: channelCalibration{Gain: 0, MinimumOutput: 80}, input: 100, want: 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.config.apply(test.input); got != test.want {
				t.Fatalf("apply(%d)=%d, want %d", test.input, got, test.want)
			}
		})
	}
}

func TestCalibrationPreservesIndependentChannels(t *testing.T) {
	config := defaultCalibration()
	config.Stick.MinimumOutput = 20
	config.Throttle.Gain = 2
	got := config.apply(motorCommand{Left: 5, Right: 40})
	if got.Left != 20 || got.Right != 80 {
		t.Fatalf("apply returned left=%d right=%d", got.Left, got.Right)
	}
}

func TestCalibrationRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	want := defaultCalibration()
	want.Stick.Gain = 1.3
	want.Stick.MinimumOutput = 25
	want.Throttle.Gain = 1.1
	want.Throttle.MinimumOutput = 30
	if err := saveCalibration(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := loadCalibration(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("round trip got %#v, want %#v", got, want)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"stick"`)) || !bytes.Contains(data, []byte(`"throttle"`)) ||
		bytes.Contains(data, []byte(`"left"`)) || bytes.Contains(data, []byte(`"right"`)) {
		t.Fatalf("saved calibration does not use semantic channel names: %s", data)
	}
}

func TestInvalidCalibrationFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(`{"version":2}`), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := loadCalibration(path)
	if err == nil {
		t.Fatal("expected invalid version error")
	}
	if got != defaultCalibration() {
		t.Fatalf("fallback got %#v", got)
	}
}

func TestLegacyLeftRightCalibrationMigrates(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	legacy := `{
  "version": 1,
  "left": {"gain": 1.2, "minimum_output": 22},
  "right": {"gain": 1.4, "minimum_output": 33},
  "reconnect_interval_ms": 1000
}`
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	config, err := loadCalibration(path)
	if err != nil {
		t.Fatal(err)
	}
	if config.Stick.Gain != 1.2 || config.Stick.MinimumOutput != 22 ||
		config.Throttle.Gain != 1.4 || config.Throttle.MinimumOutput != 33 {
		t.Fatalf("legacy migration produced %#v", config)
	}
}
