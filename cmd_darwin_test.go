package cmd

import (
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCommand_ExecuteStderr(t *testing.T) {
	cmd := New(">&2 echo hello")

	err := cmd.Run()

	assert.Nil(t, err)
	assert.Equal(t, "hello\n", cmd.Stderr())
}

func TestCommand_WithTimeout(t *testing.T) {
	cmd := New("sleep 0.5;", WithTimeout(5*time.Millisecond))

	err := cmd.Run()

	assert.NotNil(t, err)
	assert.Equal(t, "timeout after 5ms", err.Error())
}

func TestCommand_WithValidTimeout(t *testing.T) {
	cmd := New("sleep 0.01;", WithTimeout(500*time.Millisecond))

	err := cmd.Run()

	assert.Nil(t, err)
}

// I really don't see the point of mocking this
// as the stdlib does so already. So testing here
// seems redundant. This simple check if we're compliant
// with an api changes
func TestCommand_WithUser(t *testing.T) {
	cmd := New("echo hello", WithUser(syscall.Credential{Uid: 1111}))
	err := cmd.Run()
	assert.Equal(t, uint32(1111), cmd.baseCommand.SysProcAttr.Credential.Uid)
	assert.Error(t, err)
}
