//go:build !windows

package app

import (
	"fmt"
	"syscall"
)

// killSignal represents a signal to send to a process.
type killSignal int

const (
	signalTerm killSignal = killSignal(syscall.SIGTERM)
	signalKill killSignal = killSignal(syscall.SIGKILL)
)

// killProcess sends sig to the given PID.
// Rejects PID <= 1 to prevent killing init or the entire process group.
func killProcess(pid int, sig killSignal) error {
	if pid <= 1 {
		return fmt.Errorf("refusing to signal PID %d", pid)
	}
	return syscall.Kill(pid, syscall.Signal(sig))
}
