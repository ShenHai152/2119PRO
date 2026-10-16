//go:build windows

package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strings"
	"syscall"
	"time"
)

func openMotorOutput(mode string) (motorOutput, string, error) {
	switch strings.ToLower(mode) {
	case "directinput":
		output, err := newFFB()
		if err != nil {
			return nil, "", fmt.Errorf("DirectInput: %w", err)
		}
		friendlyName := syscallUTF16DeviceName()
		if friendlyName == "" {
			friendlyName = "PXN-F19"
		}
		return output, friendlyName + " connected through DirectInput/EZFRD", nil
	case "hidpath":
		output, err := newHIDMotor()
		if err != nil {
			return nil, "", fmt.Errorf("DirectInput-selected HID: %w", err)
		}
		return output, "PXN-F19 connected through DirectInput-selected native HID", nil
	default:
		return nil, "", fmt.Errorf("unknown --output %q (use directinput or hidpath)", mode)
	}
}

func syscallUTF16DeviceName() string {
	if selectedDevice == nil {
		return ""
	}
	name := syscall.UTF16ToString(selectedDevice.ProductName[:])
	if name == "" {
		name = syscall.UTF16ToString(selectedDevice.InstanceName[:])
	}
	return name
}

type consoleInput struct {
	lines <-chan string
}

func newConsoleInput() consoleInput {
	lines := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			lines <- strings.TrimSpace(scanner.Text())
		}
		close(lines)
	}()
	return consoleInput{lines: lines}
}

func (c consoleInput) wait(stop <-chan struct{}) (string, error) {
	select {
	case <-stop:
		return "", fmt.Errorf("calibration interrupted")
	case line, ok := <-c.lines:
		if !ok {
			return "", fmt.Errorf("console input closed")
		}
		return strings.ToLower(strings.TrimSpace(line)), nil
	}
}

func stopAwareWait(stop <-chan struct{}, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-stop:
		return fmt.Errorf("calibration interrupted")
	case <-timer.C:
		return nil
	}
}

func setRawCommand(output motorOutput, command motorCommand) error {
	return output.set(normalizedFloat(command.Left), normalizedFloat(command.Right))
}

func scanMotorThreshold(output motorOutput, input consoleInput, stop <-chan struct{}, channel string) (int, error) {
	defer func() { _ = output.set(0, 0) }()
	for {
		fmt.Printf("\n%s 电机起转扫描：幅度会从 5 逐步增加，首次清晰感到震动时按 Enter。\n", channel)
		selected := 0
		for level := 5; level <= 255; level += 5 {
			command := motorCommand{}
			if channel == "摇杆" {
				command.Left = byte(level)
			} else {
				command.Right = byte(level)
			}
			if err := setRawCommand(output, command); err != nil {
				return 0, err
			}
			fmt.Printf("\r%s电机原始幅度: %3d  ", channel, level)
			timer := time.NewTimer(700 * time.Millisecond)
			select {
			case <-stop:
				timer.Stop()
				return 0, fmt.Errorf("calibration interrupted")
			case _, ok := <-input.lines:
				timer.Stop()
				if !ok {
					return 0, fmt.Errorf("console input closed")
				}
				selected = level
			case <-timer.C:
			}
			if selected != 0 {
				break
			}
		}
		if err := output.set(0, 0); err != nil {
			return 0, err
		}
		fmt.Println()
		if selected == 0 {
			return 0, fmt.Errorf("%s电机在 255 幅度仍未确认起转", channel)
		}
		fmt.Printf("验证 %s 电机阈值 %d...\n", channel, selected)
		command := motorCommand{}
		if channel == "摇杆" {
			command.Left = byte(selected)
		} else {
			command.Right = byte(selected)
		}
		if err := setRawCommand(output, command); err != nil {
			return 0, err
		}
		if err := stopAwareWait(stop, time.Second); err != nil {
			return 0, err
		}
		if err := output.set(0, 0); err != nil {
			return 0, err
		}
		if err := stopAwareWait(stop, 500*time.Millisecond); err != nil {
			return 0, err
		}
		lower := selected - 5
		if lower > 0 {
			fmt.Printf("对照测试 %s 电机阈值减 5（%d）...\n", channel, lower)
			command = motorCommand{}
			if channel == "摇杆" {
				command.Left = byte(lower)
			} else {
				command.Right = byte(lower)
			}
			if err := setRawCommand(output, command); err != nil {
				return 0, err
			}
			if err := stopAwareWait(stop, time.Second); err != nil {
				return 0, err
			}
			if err := output.set(0, 0); err != nil {
				return 0, err
			}
		}
		fmt.Print("该阈值是否合适？[Enter/Y 接受，R 重试]: ")
		answer, err := input.wait(stop)
		if err != nil {
			return 0, err
		}
		if answer != "r" {
			return selected, nil
		}
	}
}

func balanceMotorGain(output motorOutput, input consoleInput, stop <-chan struct{}, config *calibrationConfig) error {
	defer func() { _ = output.set(0, 0) }()
	for {
		stick := config.Stick.apply(160)
		throttle := config.Throttle.apply(160)
		fmt.Printf("\n摇杆电机测试（校准后幅度 %d）...\n", stick)
		if err := setRawCommand(output, motorCommand{Left: stick}); err != nil {
			return err
		}
		if err := stopAwareWait(stop, 750*time.Millisecond); err != nil {
			return err
		}
		if err := output.set(0, 0); err != nil {
			return err
		}
		if err := stopAwareWait(stop, 250*time.Millisecond); err != nil {
			return err
		}
		fmt.Printf("油门电机测试（校准后幅度 %d）...\n", throttle)
		if err := setRawCommand(output, motorCommand{Right: throttle}); err != nil {
			return err
		}
		if err := stopAwareWait(stop, 750*time.Millisecond); err != nil {
			return err
		}
		if err := output.set(0, 0); err != nil {
			return err
		}
		fmt.Print("哪个电机更弱？[S 摇杆弱，T 油门弱，Enter/E 相同]: ")
		answer, err := input.wait(stop)
		if err != nil {
			return err
		}
		switch answer {
		case "s", "l":
			if config.Stick.Gain >= 4 {
				fmt.Println("摇杆增益已达到上限 4.0。")
				return nil
			}
			config.Stick.Gain = math.Min(4, math.Round((config.Stick.Gain+0.1)*10)/10)
		case "t", "r":
			if config.Throttle.Gain >= 4 {
				fmt.Println("油门增益已达到上限 4.0。")
				return nil
			}
			config.Throttle.Gain = math.Min(4, math.Round((config.Throttle.Gain+0.1)*10)/10)
		default:
			return nil
		}
	}
}

func runCalibration(output motorOutput, path string, stop <-chan struct{}) error {
	input := newConsoleInput()
	config := defaultCalibration()
	stickThreshold, err := scanMotorThreshold(output, input, stop, "摇杆")
	if err != nil {
		return err
	}
	config.Stick.MinimumOutput = stickThreshold
	throttleThreshold, err := scanMotorThreshold(output, input, stop, "油门")
	if err != nil {
		return err
	}
	config.Throttle.MinimumOutput = throttleThreshold
	if err := balanceMotorGain(output, input, stop, &config); err != nil {
		return err
	}
	fmt.Printf("\n校准结果：摇杆 gain=%.1f min=%d，油门 gain=%.1f min=%d\n",
		config.Stick.Gain, config.Stick.MinimumOutput,
		config.Throttle.Gain, config.Throttle.MinimumOutput)
	fmt.Printf("保存到 %s？[Enter/Y 保存，N 取消]: ", path)
	answer, err := input.wait(stop)
	if err != nil {
		return err
	}
	if answer == "n" {
		fmt.Println("校准已取消，现有配置未修改。")
		return nil
	}
	if err := saveCalibration(path, config); err != nil {
		return err
	}
	fmt.Println("校准配置已保存。")
	return nil
}
