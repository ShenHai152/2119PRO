package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

const calibrationVersion = 1

type channelCalibration struct {
	Gain          float64 `json:"gain"`
	MinimumOutput int     `json:"minimum_output"`
}

type calibrationConfig struct {
	Version             int                `json:"version"`
	Stick               channelCalibration `json:"stick"`
	Throttle            channelCalibration `json:"throttle"`
	ReconnectIntervalMS int                `json:"reconnect_interval_ms"`
}

type calibrationFile struct {
	Version             int                 `json:"version"`
	Stick               *channelCalibration `json:"stick"`
	Throttle            *channelCalibration `json:"throttle"`
	Left                *channelCalibration `json:"left"`
	Right               *channelCalibration `json:"right"`
	ReconnectIntervalMS int                 `json:"reconnect_interval_ms"`
}

func defaultCalibration() calibrationConfig {
	return calibrationConfig{
		Version: calibrationVersion,
		Stick: channelCalibration{
			Gain: 1,
		},
		Throttle: channelCalibration{
			Gain: 1,
		},
		ReconnectIntervalMS: 1000,
	}
}

func (c calibrationConfig) validate() error {
	if c.Version != calibrationVersion {
		return fmt.Errorf("unsupported calibration version %d", c.Version)
	}
	if err := c.Stick.validate("stick"); err != nil {
		return err
	}
	if err := c.Throttle.validate("throttle"); err != nil {
		return err
	}
	if c.ReconnectIntervalMS < 100 || c.ReconnectIntervalMS > 60000 {
		return fmt.Errorf("reconnect_interval_ms must be within 100..60000")
	}
	return nil
}

func (c channelCalibration) validate(name string) error {
	if math.IsNaN(c.Gain) || math.IsInf(c.Gain, 0) || c.Gain < 0 || c.Gain > 4 {
		return fmt.Errorf("%s gain must be within 0..4", name)
	}
	if c.MinimumOutput < 0 || c.MinimumOutput > 255 {
		return fmt.Errorf("%s minimum_output must be within 0..255", name)
	}
	return nil
}

func (c channelCalibration) apply(input byte) byte {
	if input == 0 || c.Gain == 0 {
		return 0
	}
	value := int(math.Round(float64(input) * c.Gain))
	if value < 1 {
		value = 1
	}
	if value < c.MinimumOutput {
		value = c.MinimumOutput
	}
	if value > 255 {
		value = 255
	}
	return byte(value)
}

func (c calibrationConfig) apply(command motorCommand) motorCommand {
	return motorCommand{
		Left:  c.Stick.apply(command.Left),
		Right: c.Throttle.apply(command.Right),
	}
}

func defaultCalibrationPath() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		var err error
		base, err = os.UserConfigDir()
		if err != nil {
			return "", fmt.Errorf("locate user configuration directory: %w", err)
		}
	}
	return filepath.Join(base, "PXN2119FFBBridge", "config.json"), nil
}

func loadCalibration(path string) (calibrationConfig, error) {
	defaults := defaultCalibration()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return defaults, nil
	}
	if err != nil {
		return defaults, err
	}
	var file calibrationFile
	if err := json.Unmarshal(data, &file); err != nil {
		return defaults, err
	}
	if file.Stick == nil {
		file.Stick = file.Left
	}
	if file.Throttle == nil {
		file.Throttle = file.Right
	}
	if file.Stick == nil || file.Throttle == nil {
		return defaults, fmt.Errorf("calibration requires stick and throttle channels")
	}
	config := calibrationConfig{
		Version:             file.Version,
		Stick:               *file.Stick,
		Throttle:            *file.Throttle,
		ReconnectIntervalMS: file.ReconnectIntervalMS,
	}
	if err := config.validate(); err != nil {
		return defaults, err
	}
	return config, nil
}

func saveCalibration(path string, config calibrationConfig) error {
	if err := config.validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create calibration directory: %w", err)
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), "config-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary calibration: %w", err)
	}
	temporaryPath := temporary.Name()
	removeTemporary := true
	defer func() {
		if removeTemporary {
			_ = os.Remove(temporaryPath)
		}
	}()
	encoder := json.NewEncoder(temporary)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(config); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("encode calibration: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("flush calibration: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close calibration: %w", err)
	}
	if err := replaceFile(temporaryPath, path); err != nil {
		return fmt.Errorf("install calibration: %w", err)
	}
	removeTemporary = false
	return nil
}

func normalizedByte(value float64) byte {
	if value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return byte(math.Round(value * 255))
}

func normalizedFloat(value byte) float64 {
	return float64(value) / 255
}

type calibratedMotor struct {
	raw    motorOutput
	config calibrationConfig
}

func (m *calibratedMotor) set(left, right float64) error {
	command := m.config.apply(motorCommand{
		Left:  normalizedByte(left),
		Right: normalizedByte(right),
	})
	return m.raw.set(normalizedFloat(command.Left), normalizedFloat(command.Right))
}

func (m *calibratedMotor) healthy() error {
	if m == nil || m.raw == nil {
		return errMotorDisconnected
	}
	return m.raw.healthy()
}

func (m *calibratedMotor) close() {
	if m != nil && m.raw != nil {
		m.raw.close()
	}
}
