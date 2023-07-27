package cmd

import (
	"os/exec"
	"syscall"
)

func createBaseCommand(c *Cmd) *exec.Cmd {
	return exec.Command("/bin/sh", "-c", c.Command)
}

// WithUser allows the command to be run as a different
// user.
//
// Example:
//
//	cred := syscall.Credential{Uid: 1000, Gid: 1000}
//	c := New("echo hello", cred)
//	c.Run()
func WithUser(credential syscall.Credential) func(c *Cmd) {
	return func(c *Cmd) {
		c.baseCommand.SysProcAttr = &syscall.SysProcAttr{
			Credential: &credential,
		}
	}
}
