package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

func main() {
	fmt.Print("Stub main")
}

// RunResult the result of the run function
type RunResult struct {
	exitCode int
	stdout   string
	stderr   string
	duration time.Duration
}

// Run a command
func Run(cmd *exec.Cmd) RunResult {
	var result RunResult
	var outbuf, errbuf bytes.Buffer
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	start := time.Now()

	err := cmd.Run()

	result.duration = time.Since(start)

	result.stdout = outbuf.String()
	result.stderr = errbuf.String()

	result.exitCode = 0
	if err != nil {
		result.exitCode = 999

		var e *exec.ExitError
		if errors.As(err, &e) {
			result.exitCode = e.ExitCode()
		}
	}

	return result
}
