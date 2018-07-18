// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package taskmanager

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"

	"github.com/Azure/acr-builder/util"
)

// TaskManager is a wrapper for executing processes.
type TaskManager struct {
	DryRun    bool
	mu        sync.Mutex
	processes map[int]*os.Process
}

// NewTaskManager creates a new TaskManager.
func NewTaskManager(dryRun bool) *TaskManager {
	return &TaskManager{
		DryRun:    dryRun,
		processes: map[int]*os.Process{},
		mu:        sync.Mutex{},
	}
}

// Run runs a command based on the specified params.
func (tm *TaskManager) Run(
	ctx context.Context,
	args []string,
	stdIn io.Reader,
	stdOut io.Writer,
	stdErr io.Writer,
	cmdDir string) error {

	if tm.DryRun {
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

	tm.mu.Lock()
	tm.processes[pid] = cmd.Process
	tm.mu.Unlock()

	defer tm.DeletePid(pid)

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
func (tm *TaskManager) DeletePid(pid int) {
	tm.mu.Lock()
	delete(tm.processes, pid)
	tm.mu.Unlock()
}

// Stop stops the task manager and tries to kill any running processes.
func (tm *TaskManager) Stop() util.Errors {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	var errors util.Errors
	for pid, process := range tm.processes {
		if err := process.Kill(); err != nil {
			errors = append(errors, err)
		}

		delete(tm.processes, pid)
	}

	return errors
}
