//go:build mage
// +build mage

package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// binary name derived from project.
const binName = "specto"

// cmdPath is the import path for the main CLI package.
const cmdPath = "./cmd/specto"

// deploy defaults — override with environment variables.
var (
	deployHost = envOr("DEPLOY_HOST", "prod.example.com")
	deployUser = envOr("DEPLOY_USER", "specto")
	deployDir  = envOr("DEPLOY_DIR", "/opt/specto")
	deployUnit = envOr("DEPLOY_UNIT", "specto")
)

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

// runEnv executes a command with extra environment variables.
func runEnv(env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Deploy builds a linux/amd64 production binary, rsyncs it to the remote
// server and restarts the systemd unit.
func Deploy() error {
	linuxBin := "bin/" + binName

	// 1. Cross-compile for linux/amd64.
	fmt.Println("▶ Cross-compiling for linux/amd64…")
	if err := os.MkdirAll("bin", 0o755); err != nil {
		return err
	}
	if err := runEnv(
		[]string{"GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0"},
		"go", "build", "-ldflags", "-s -w", "-trimpath", "-o", linuxBin, cmdPath,
	); err != nil {
		return fmt.Errorf("build: %w", err)
	}

	// 2. Rsync binary + unit file to the remote host.
	dest := deployUser + "@" + deployHost
	fmt.Printf("▶ Uploading to %s:%s…\n", dest, deployDir)
	if err := run("rsync", "-avz", "--progress",
		linuxBin, "deploy/specto.service",
		dest+":"+deployDir+"/",
	); err != nil {
		return fmt.Errorf("rsync: %w", err)
	}

	// 3. Restart the systemd service.
	fmt.Println("▶ Restarting systemd unit…")
	remote := strings.Join([]string{
		"sudo cp " + deployDir + "/specto.service /etc/systemd/system/" + deployUnit + ".service",
		"sudo systemctl daemon-reload",
		"sudo systemctl enable " + deployUnit,
		"sudo systemctl restart " + deployUnit,
		"sudo systemctl status " + deployUnit + " --no-pager",
	}, " && ")

	if err := run("ssh", dest, remote); err != nil {
		return fmt.Errorf("remote restart: %w", err)
	}

	fmt.Println("✔ Deploy complete")
	return nil
}

// envOr returns the value of the environment variable, or the fallback.
func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
