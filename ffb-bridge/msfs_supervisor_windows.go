//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"pxn2119-ffb-bridge/internal/winproc"
)

const (
	msfsProcessScanInterval = 500 * time.Millisecond
	msfsAdapterRestartDelay = 2 * time.Second
)

type msfsAdapterWait struct {
	err error
}

type msfsAdapterSupervisor struct {
	path             string
	serviceAddress   string
	telemetryEnabled bool
	telemetryGain    float64
	command          *exec.Cmd
	wait             chan msfsAdapterWait
	nextScan         time.Time
	nextStart        time.Time
	missingLogged    bool
}

func newMSFSAdapterSupervisor(setting, serviceAddress string, telemetryEnabled bool, telemetryGain float64) (*msfsAdapterSupervisor, error) {
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate bridge executable: %w", err)
	}
	path, enabled, err := resolveMSFSAdapterPath(setting, executable)
	if err != nil || !enabled {
		return nil, err
	}
	return &msfsAdapterSupervisor{
		path: path, serviceAddress: serviceAddress,
		telemetryEnabled: telemetryEnabled, telemetryGain: telemetryGain,
	}, nil
}

func (s *msfsAdapterSupervisor) start(stop <-chan struct{}) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(msfsProcessScanInterval)
		defer ticker.Stop()
		s.tick(time.Now())
		for {
			select {
			case <-stop:
				s.closeChild()
				return
			case now := <-ticker.C:
				s.tick(now)
			}
		}
	}()
	return done
}

func (s *msfsAdapterSupervisor) tick(now time.Time) {
	if s.command != nil {
		select {
		case result := <-s.wait:
			if result.err != nil {
				fmt.Fprintf(os.Stderr, "MSFS adapter exited: %v\n", result.err)
			} else {
				fmt.Println("MSFS adapter exited with the simulator")
			}
			s.command = nil
			s.wait = nil
			s.nextStart = now.Add(msfsAdapterRestartDelay)
		default:
		}
	}
	if now.Before(s.nextScan) {
		return
	}
	s.nextScan = now.Add(msfsProcessScanInterval)
	process, running, err := winproc.FindMSFS()
	if err != nil {
		fmt.Fprintf(os.Stderr, "MSFS process detection failed: %v\n", err)
		return
	}
	if !running || s.command != nil || now.Before(s.nextStart) {
		return
	}
	if _, err := os.Stat(s.path); err != nil {
		if !s.missingLogged {
			fmt.Fprintf(os.Stderr, "MSFS adapter auto-start unavailable: %s (%v)\n", s.path, err)
			s.missingLogged = true
		}
		return
	}

	command := exec.Command(s.path,
		"--service", s.serviceAddress,
		"--parent-pid", strconv.Itoa(os.Getpid()),
		"--telemetry="+strconv.FormatBool(s.telemetryEnabled),
		"--telemetry-gain", strconv.FormatFloat(s.telemetryGain, 'f', 3, 64),
	)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := command.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "start MSFS adapter %s: %v\n", s.path, err)
		s.nextStart = now.Add(msfsAdapterRestartDelay)
		return
	}
	label, _ := winproc.ClassifyMSFSExecutable(process.Name)
	fmt.Printf("MSFS detected: %s (pid=%d); adapter started\n", label, process.PID)
	s.command = command
	wait := make(chan msfsAdapterWait, 1)
	s.wait = wait
	go func() {
		wait <- msfsAdapterWait{err: command.Wait()}
	}()
}

func (s *msfsAdapterSupervisor) closeChild() {
	if s == nil || s.command == nil || s.command.Process == nil {
		return
	}
	// The bridge itself synchronously clears both motor channels before exit.
	// The adapter also watches the parent PID and normally exits on its own;
	// Kill is only the bounded shutdown fallback for the child process.
	_ = s.command.Process.Kill()
}
