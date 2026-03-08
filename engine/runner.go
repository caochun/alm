package engine

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CommandRunner abstracts command execution so the engine can be tested
// without running real shell commands.
type CommandRunner interface {
	Run(ctx context.Context, cmd string, args []string, env map[string]string) (output string, err error)
}

// ShellRunner executes commands via the system shell.
type ShellRunner struct{}

func (ShellRunner) Run(ctx context.Context, cmd string, args []string, env map[string]string) (string, error) {
	c := exec.CommandContext(ctx, cmd, args...)
	if len(env) > 0 {
		for k, v := range env {
			c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return stdout.String(), fmt.Errorf("%w: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// DryRunRunner prints commands instead of executing them.
type DryRunRunner struct {
	Log []string
}

func (d *DryRunRunner) Run(_ context.Context, cmd string, args []string, env map[string]string) (string, error) {
	line := fmt.Sprintf("[dry-run] %s %s", cmd, strings.Join(args, " "))
	if len(env) > 0 {
		envPairs := make([]string, 0, len(env))
		for k, v := range env {
			envPairs = append(envPairs, fmt.Sprintf("%s=%s", k, v))
		}
		line = fmt.Sprintf("[dry-run] env %s %s %s", strings.Join(envPairs, " "), cmd, strings.Join(args, " "))
	}
	d.Log = append(d.Log, line)
	return fmt.Sprintf("(dry-run) would execute: %s", cmd), nil
}
