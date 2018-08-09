// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package procmanager

import (
	"context"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/Azure/acr-builder/pkg/util"
)

// ProcManager is a wrapper for os.Process.
type ProcManager struct {
	DryRun    bool
	mu        sync.Mutex
	processes map[int]*os.Process
}

// NewProcManager creates a new ProcManager.
func NewProcManager(dryRun bool) *ProcManager {
	return &ProcManager{
		DryRun:    dryRun,
		processes: map[int]*os.Process{},
		mu:        sync.Mutex{},
	}
}

// Run runs an exec.Command based on the specified args.
// stdIn, stdOut, stdErr, and cmdDir can be attached to the created exec.Command.
func (pm *ProcManager) Run(
	ctx context.Context,
	args []string,
	stdIn io.Reader,
	stdOut io.Writer,
	stdErr io.Writer,
	cmdDir string) error {
	if pm.DryRun {
		log.Printf("[DRY RUN] Args: %v\n", args)
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

	pm.mu.Lock()
	pm.processes[pid] = cmd.Process
	pm.mu.Unlock()

	defer pm.DeletePid(pid)

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
				log.Printf("Failed to kill process. Path: %s, Args: %v, Err: %v", cmd.Path, cmd.Args, err)
			}
		}()

		return ctx.Err()
	}
}

// DeletePid deletes the specified pid from the internal map.
func (pm *ProcManager) DeletePid(pid int) {
	pm.mu.Lock()
	delete(pm.processes, pid)
	pm.mu.Unlock()
}

// Stop stops the process manager and tries to kill any remaining processes
// in its internal map. Any errors encountered during kill will be return as
// a list of errors.
func (pm *ProcManager) Stop() util.Errors {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors util.Errors
	for pid, process := range pm.processes {
		if err := process.Kill(); err != nil {
			errors = append(errors, err)
		}
		delete(pm.processes, pid)
	}
	return errors
}
