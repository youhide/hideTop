//go:build windows

package app

import (
	"fmt"
	"os/exec"
)

// killSignal represents a signal to send to a process.
type killSignal int

const (
	signalTerm killSignal = 15
	signalKill killSignal = 9
)

// killProcess terminates the given PID on Windows via taskkill.
func killProcess(pid int, sig killSignal) error {
	var cmd *exec.Cmd
	if sig == signalKill {
		cmd = exec.Command("taskkill", "/F", "/PID", fmt.Sprint(pid))
	} else {
		cmd = exec.Command("taskkill", "/PID", fmt.Sprint(pid))
	}
	return cmd.Run()
}
