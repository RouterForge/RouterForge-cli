package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

type ShellResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

type ShellRunner struct {
	workdir string
}

func NewShellRunner(workdir string) *ShellRunner {
	return &ShellRunner{workdir: workdir}
}

func (sr *ShellRunner) Run(ctx context.Context, command string) (*ShellResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = sr.workdir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run command: %w", err)
		}
	}

	return &ShellResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

func (sr *ShellRunner) RunWithInput(ctx context.Context, command string, input string) (*ShellResult, error) {
	start := time.Now()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = sr.workdir
	cmd.Stdin = strings.NewReader(input)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("failed to run command: %w", err)
		}
	}

	return &ShellResult{
		Stdout:   strings.TrimSpace(stdout.String()),
		Stderr:   strings.TrimSpace(stderr.String()),
		ExitCode: exitCode,
		Duration: duration,
	}, nil
}

func (sr *ShellRunner) InstallGoDeps(goBin string) error {
	cmd := exec.Command("bash", "-c", fmt.Sprintf("export PATH=\"%s:$PATH\" && cd %s && go mod tidy", goBin, sr.workdir))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
