package cmd_test

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/bingoohuang/cmd"
	"github.com/stretchr/testify/assert"
)

func TestCommand_ExecuteStderr(t *testing.T) {
	c := cmd.New("echo hello 1>&2")
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assertEqualWithLineBreak(t, "hello ", c.Stderr())
}

func TestCommand_WithTimeout(t *testing.T) {
	c := cmd.New("timeout 0.005;", cmd.WithTimeout(5*time.Millisecond))
	err := c.Run(context.TODO())

	assert.NotNil(t, err)
	// This is needed because windows sometimes can not kill the process :(
	containsMsg := strings.Contains(err.Error(), "timeout, kill") || strings.Contains(err.Error(), "timeout after 5ms")
	assert.True(t, containsMsg)
}

func TestCommand_WithValidTimeout(t *testing.T) {
	c := cmd.New("timeout 0.01;", cmd.WithTimeout(1000*time.Millisecond))
	err := c.Run(context.TODO())

	assert.Nil(t, err)
}

func TestCommand_WithUser(t *testing.T) {
	onehundred := 100
	token := syscall.Token(uintptr(unsafe.Pointer(&onehundred)))
	c := cmd.New("echo hello", cmd.WithUser(token))
	err := c.Run(context.TODO())
	assert.Error(t, err)
}
