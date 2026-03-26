//go:build !windows

package app

import "syscall"

// killSignal represents a signal to send to a process.
type killSignal int

const (
	signalTerm killSignal = killSignal(syscall.SIGTERM)
	signalKill killSignal = killSignal(syscall.SIGKILL)
)

// killProcess sends sig to the given PID.
func killProcess(pid int, sig killSignal) error {
	return syscall.Kill(pid, syscall.Signal(sig))
}
