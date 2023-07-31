package gocmd_test

import (
	"context"
	"strings"
	"syscall"
	"testing"
	"time"
	"unsafe"

	"github.com/bingoohuang/gocmd"
	"github.com/stretchr/testify/assert"
)

func TestCommand_ExecuteStderr(t *testing.T) {
	c := gocmd.New("echo hello 1>&2")
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assertEqualWithLineBreak(t, "hello ", c.Stderr())
}

func TestCommand_WithTimeout(t *testing.T) {
	c := gocmd.New("timeout 0.005;", gocmd.WithTimeout(5*time.Millisecond))
	err := c.Run(context.TODO())

	assert.NotNil(t, err)
	// This is needed because windows sometimes can not kill the process :(
	containsMsg := strings.Contains(err.Error(), "timeout, kill") || strings.Contains(err.Error(), "timeout after 5ms")
	assert.True(t, containsMsg)
}

func TestCommand_WithValidTimeout(t *testing.T) {
	c := gocmd.New("timeout 0.01;", gocmd.WithTimeout(1000*time.Millisecond))
	err := c.Run(context.TODO())

	assert.Nil(t, err)
}

func TestCommand_WithUser(t *testing.T) {
	onehundred := 100
	token := syscall.Token(uintptr(unsafe.Pointer(&onehundred)))
	c := gocmd.New("echo hello", gocmd.WithUser(token))
	err := c.Run(context.TODO())
	assert.Error(t, err)
}
