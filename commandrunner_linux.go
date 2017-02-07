package cgsetup

// Original source: https://github.com/cloudfoundry/commandrunner/blob/master/linux_command_runner/linux_command_runner.go
// code has been modified slightly (renamed the "New" func)

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

type RealCommandRunner struct{}

type CommandNotRunningError struct {
	cmd *exec.Cmd
}

func (e CommandNotRunningError) Error() string {
	return fmt.Sprintf("command is not running: %#v", e.cmd)
}

func NewRealCommandRunner() *RealCommandRunner {
	return &RealCommandRunner{}
}

func (r *RealCommandRunner) Run(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	return cmd.Run()
}

func (r *RealCommandRunner) Start(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	return cmd.Start()
}

func (r *RealCommandRunner) Background(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	return cmd.Start()
}

func (r *RealCommandRunner) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}

func (r *RealCommandRunner) Kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return CommandNotRunningError{cmd}
	}

	return cmd.Process.Kill()
}

func (r *RealCommandRunner) Signal(cmd *exec.Cmd, signal os.Signal) error {
	if cmd.Process == nil {
		return CommandNotRunningError{cmd}
	}

	return cmd.Process.Signal(signal)
}
