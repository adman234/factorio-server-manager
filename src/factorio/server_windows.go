package factorio

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

func sendCtrlCToPid(pid int) error {
	d, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return fmt.Errorf("LoadDLL: %w", err)
	}
	p, err := d.FindProc("GenerateConsoleCtrlEvent")
	if err != nil {
		return fmt.Errorf("FindProc: %w", err)
	}
	r, _, err := p.Call(uintptr(syscall.CTRL_C_EVENT), uintptr(pid))
	if r == 0 {
		return fmt.Errorf("GenerateConsoleCtrlEvent: %w", err)
	}
	return nil
}

func setCtrlHandlingIsDisabledForThisProcess(disabled bool) error {
	disabledInt := 0
	if disabled {
		disabledInt = 1
	}

	d, err := syscall.LoadDLL("kernel32.dll")
	if err != nil {
		return fmt.Errorf("LoadDLL: %w", err)
	}
	p, err := d.FindProc("SetConsoleCtrlHandler")
	if err != nil {
		return fmt.Errorf("FindProc: %w", err)
	}
	r, _, err := p.Call(uintptr(0), uintptr(disabledInt))
	if r == 0 {
		return fmt.Errorf("SetConsoleCtrlHandler: %w", err)
	}
	return nil
}

// interruptProcess sends CTRL+C to the console, which makes Factorio save the map.
func interruptProcess(cmd *exec.Cmd) error {
	// Disable our own handling of CTRL+C, so we don't close when we send it to the console.
	if err := setCtrlHandlingIsDisabledForThisProcess(true); err != nil {
		return err
	}
	// Re-enable handling of CTRL+C after the event has been delivered.
	defer func() {
		time.Sleep(20 * time.Millisecond)
		if err := setCtrlHandlingIsDisabledForThisProcess(false); err != nil {
			log.Printf("could not re-enable CTRL+C handling: %s", err)
		}
	}()

	// Send CTRL+C to all processes attached to the console (ourself, and the factorio server instance)
	return sendCtrlCToPid(0)
}

func (server *Server) checkProcessHealth(text string) {
	// check if the output indicates a server shutdown
	if strings.Contains(text, "ServerMultiplayerManager.cpp:783: updateTick(0) changing state from(Disconnected) to(Closed)") {
		// Somehow, the Factorio devs managed to code the game to react appropriately to CTRL+C, including
		// saving the game, but not actually exit. So, we still have to manually kill the process, and
		// for extra fun, there's no way to know when the server save has actually completed (unless we want
		// to inject filesystem logic into what should be a process-level Stop() routine), so our best option
		// is to just wait an arbitrary amount of time and hope that the save is successful in that time.
		if cmd, _ := server.runningCmd(); cmd != nil {
			cmd.Process.Kill()
		}
	}
}
