package cmder

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/Azure/acr-builder/util"
)

// ICmder is an interface for executing processes.
type ICmder interface {
	// Run runs a command based on the specified params.
	Run(ctx context.Context,
		args []string,
		stdIn io.Reader,
		stdOut io.Writer,
		stdErr io.Writer,
		cmdDir string) error

	// Stop stops the runner and tries to kill any running processes.
	Stop() error
}

// Cmder is a wrapper for executing processes.
type Cmder struct {
	DryRun    bool
	mu        sync.Mutex
	processes map[int]*os.Process
}

// NewCmder creates a new Cmder.
func NewCmder(dryRun bool) *Cmder {
	return &Cmder{
		DryRun:    dryRun,
		processes: map[int]*os.Process{},
		mu:        sync.Mutex{},
	}
}

// Run runs a command based on the specified params.
func (c *Cmder) Run(
	ctx context.Context,
	args []string,
	stdIn io.Reader,
	stdOut io.Writer,
	stdErr io.Writer,
	cmdDir string) error {

	if c.DryRun {
		fmt.Printf("[DRY RUN] Args: %v\n", args)
		return nil
	}

	cmd := exec.Command(args[0], args[1:]...)
	if cmdDir != "" {
		cmd.Dir = cmdDir
	}

	cmd.Stdin = stdIn
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	if err := cmd.Start(); err != nil {
		return err
	}

	pid := cmd.Process.Pid

	c.mu.Lock()
	c.processes[pid] = cmd.Process
	c.mu.Unlock()

	defer c.DeletePid(pid)

	errChan := make(chan error)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case err := <-errChan:
		return err

	case <-ctx.Done():
		go func() {
			if err := cmd.Process.Kill(); err != nil {
				fmt.Printf("Failed to kill process. Path: %v, Args: %v, Err: %v", cmd.Path, cmd.Args, err)
			}
		}()

		return ctx.Err()
	}
}

// DeletePid deletes the specified pid from the internal map.
func (c *Cmder) DeletePid(pid int) {
	c.mu.Lock()
	delete(c.processes, pid)
	c.mu.Unlock()
}

// Stop stops the runner and tries to kill any running processes.
func (c *Cmder) Stop() util.Errors {
	c.mu.Lock()
	defer c.mu.Unlock()

	var errors util.Errors
	for pid, process := range c.processes {
		if err := process.Kill(); err != nil {
			errors = append(errors, err)
		}

		delete(c.processes, pid)
	}

	return errors
}
