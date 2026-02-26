//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// binary name derived from project.
const binName = "specto"

// cmdPath is the import path for the main CLI package.
const cmdPath = "./cmd/specto"

// Dev runs the Specto server in development mode.
func Dev() error {
	fmt.Println("▶ Starting dev server…")
	return run("go", "run", cmdPath, "server")
}

// Prod builds an optimised, stripped production binary into ./bin/.
func Prod() error {
	fmt.Println("▶ Building production binary…")
	output := "bin/" + binName
	if runtime.GOOS == "windows" {
		output += ".exe"
	}

	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}

	return run("go", "build",
		"-ldflags", "-s -w",
		"-trimpath",
		"-o", output,
		cmdPath,
	)
}

// Test runs all project tests.
func Test() error {
	fmt.Println("▶ Running tests…")
	return run("go", "test", "-count=1", "./...")
}

// Seed invokes the specto seed command (must be built first or use go run).
func Seed() error {
	fmt.Println("▶ Running seed…")
	return run("go", "run", cmdPath, "seed")
}

// run executes a command, forwarding stdout/stderr to the terminal.
func run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
