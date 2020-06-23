package main

import (
	"fmt"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

var commandTests = []struct {
	program          *exec.Cmd
	expectedExitCode int
	expectedStdOut   string
	expectedStdErr   string
}{
	{exec.Command("test/resources/helloworld.py"), 0, "hello world!\n", ""},
	{exec.Command("test/resources/ok_print_arguments.py", "hi", "there"), 0, "hi there\n", ""},
	{exec.Command("test/resources/error_print_arguments.py", "oooops", "something wrong"), 1, "", "oooops something wrong\n"},
	{exec.Command("test/resources/error_exit2.py"), 2, "", ""},
	{exec.Command("test/resources/not_existing.py"), 999, "", ""},
}

func TestCommands(t *testing.T) {
	for _, tt := range commandTests {
		assert := assert.New(t)

		result := Run(tt.program)

		assert.Equalf(tt.expectedExitCode, result.exitCode, "Exit code incorrect for `%s`", tt.program)
		assert.Equalf(tt.expectedStdOut, result.stdout, "StdOut has content check failed for `%s`", tt.program)
		assert.Equalf(tt.expectedStdErr, result.stderr, "StdErr has content check failed for `%s`", tt.program)
	}
}

func TestDuration(t *testing.T) {
	assert := assert.New(t)

	duration := 2.0
	program := exec.Command("test/resources/sleeping.py", fmt.Sprintf("%f", duration))
	result := Run(program)

	assert.Equal(0, result.exitCode, "Exit code incorrect for `%s`", program)
	assert.GreaterOrEqual(result.duration.Seconds(), duration, "Duration incorrect `%s`", program)
	assert.LessOrEqual(result.duration.Seconds(), duration+1, "Duration incorrect `%s`", program)
}
