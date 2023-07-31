//go:build !windows

package gocmd

import (
	"os/exec"
	"syscall"
)

func createBaseCommand(c *Cmd) *exec.Cmd {
	return exec.Command("/bin/bash", "-c", c.Command)
}

// WithUser allows the command to be run as a different
// user.
//
// Example:
//
//	cred := syscall.Credential{Uid: 1000, Gid: 1000}
//	c := New("echo hello", cred)
//	c.Run(context.TODO())
func WithUser(credential syscall.Credential) func(c *Cmd) {
	return func(c *Cmd) {
		if c.Cmd.SysProcAttr == nil {
			c.Cmd.SysProcAttr = &syscall.SysProcAttr{}
		}
		c.Cmd.SysProcAttr.Credential = &credential
	}
}
