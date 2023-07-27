package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/bingoohuang/cmd"
	"github.com/stretchr/testify/assert"
)

func TestCommand_NewCommand(t *testing.T) {
	c := cmd.New("echo hello")
	c.Run(context.TODO())

	assertEqualWithLineBreak(t, "hello", c.Combined())
	assertEqualWithLineBreak(t, "hello", c.Stdout())
}

func TestCommand_Execute(t *testing.T) {
	c := cmd.New("echo hello")

	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.True(t, c.Executed)
	assertEqualWithLineBreak(t, "hello", c.Stdout())
}

func TestCommand_ExitCode(t *testing.T) {
	c := cmd.New("exit 120")

	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.Equal(t, 120, c.ExitCode())
}

func TestCommand_WithEnvVariables(t *testing.T) {
	envVar := "$TEST"
	if runtime.GOOS == "windows" {
		envVar = "%TEST%"
	}
	c := cmd.New(fmt.Sprintf("echo %s", envVar))
	c.Env = []string{"TEST=hey"}

	_ = c.Run(context.TODO())

	assertEqualWithLineBreak(t, "hey", c.Stdout())
}

func TestCommand_Executed(t *testing.T) {
	defer func() {
		r := recover()
		if r != nil {
			assert.Contains(t, r, "Can not read Stdout if command was not Executed")
		}
		assert.NotNil(t, r)
	}()

	c := cmd.New("echo will not be Executed")
	_ = c.Stdout()
}

func TestCommand_AddEnv(t *testing.T) {
	c := cmd.New("echo test", cmd.WithoutEnv())
	c.AddEnv("key", "value")
	assert.Equal(t, []string{"key=value"}, c.Env)
}

func TestCommand_AddEnvWithShellVariable(t *testing.T) {
	const TestEnvKey = "COMMANDER_TEST_SOME_KEY"
	os.Setenv(TestEnvKey, "test from shell")
	defer os.Unsetenv(TestEnvKey)

	c := cmd.New(getCommand())
	c.AddEnv("SOME_KEY", fmt.Sprintf("${%s}", TestEnvKey))

	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assertEqualWithLineBreak(t, "test from shell", c.Stdout())
}

func TestCommand_AddMultipleEnvWithShellVariable(t *testing.T) {
	const TestEnvKeyPlanet = "CMD_TEST_PLANET"
	const TestEnvKeyName = "CMD_TEST_NAME"
	os.Setenv(TestEnvKeyPlanet, "world")
	os.Setenv(TestEnvKeyName, "Simon")
	defer func() {
		os.Unsetenv(TestEnvKeyPlanet)
		os.Unsetenv(TestEnvKeyName)
	}()

	c := cmd.New(getCommand())
	envValue := fmt.Sprintf("Hello ${%s}, I am ${%s}", TestEnvKeyPlanet, TestEnvKeyName)
	c.AddEnv("SOME_KEY", envValue)

	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assertEqualWithLineBreak(t, "Hello world, I am Simon", c.Stdout())
}

func getCommand() string {
	command := "echo $SOME_KEY"
	if runtime.GOOS == "windows" {
		command = "echo %SOME_KEY%"
	}
	return command
}

func TestCommand_SetOptions(t *testing.T) {
	writer := &bytes.Buffer{}

	setWriter := func(c *cmd.Cmd) {
		c.StdoutWriter = writer
	}
	setTimeout := func(c *cmd.Cmd) {
		c.Timeout = 1 * time.Second
	}

	c := cmd.New("echo test", setTimeout, setWriter)
	err := c.Run(context.TODO())

	assert.Nil(t, err)
	assert.Equal(t, time.Duration(1000000000), c.Timeout)
	assertEqualWithLineBreak(t, "test", writer.String())
}

func assertEqualWithLineBreak(t *testing.T, expected string, actual string) {
	if runtime.GOOS == "windows" {
		expected += "\r\n"
	} else {
		expected += "\n"
	}

	assert.Equal(t, expected, actual)
}
