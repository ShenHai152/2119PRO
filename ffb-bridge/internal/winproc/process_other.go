//go:build !windows

package winproc

type Process struct {
	PID  uint32
	Name string
}

func FindMSFS() (Process, bool, error) { return Process{}, false, nil }

func Alive(uint32) (bool, error) { return false, nil }
