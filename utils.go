package main

/**
 * Utility methods/classes for sysdash
 */

import (
	"bytes"
	"os/exec"
	"regexp"
	"syscall"
)

////////////////////////////////////////////
// Utility: Formatting
////////////////////////////////////////////

var ANSI_REGEXP = regexp.MustCompile(`\x1B\[(([0-9]{1,2})?(;)?([0-9]{1,2})?)?[m,K,H,f,J]`)

func stripANSI(str string) string {
	return ANSI_REGEXP.ReplaceAllLiteralString(str, "")
}

////////////////////////////////////////////
// Utility: Command Exec
////////////////////////////////////////////

func execAndGetOutput(name string, workingDirectory *string, args ...string) (stdout string, exitCode int, err error) {
	cmd := exec.Command(name, args...)

	var out bytes.Buffer
	cmd.Stdout = &out

	if workingDirectory != nil {
		cmd.Dir = *workingDirectory
	}

	err = cmd.Run()

	// Getting the exit code is platform dependant, this code isn't portable
	exitCode = 0

	if err != nil {
		// Based on: https://stackoverflow.com/questions/10385551/get-exit-code-go
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// Failed, but on a platform where this conversion doesn't work...
			exitCode = 1
		}
	}

	stdout = out.String()

	return
}
