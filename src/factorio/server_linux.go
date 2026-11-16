//go:build !windows

package factorio

import (
	"os"
	"os/exec"
)

// interruptProcess sends SIGINT, which makes Factorio save the map and exit.
func interruptProcess(cmd *exec.Cmd) error {
	return cmd.Process.Signal(os.Interrupt)
}

func (server *Server) checkProcessHealth(text string) {
	// ignore
}
