// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package procmanager

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"time"

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

// RunRepeatWithRetries performs a Run multiple times with retries.
// If any error occurs during the repetition, all errors will be aggregated and returned.
func (pm *ProcManager) RunRepeatWithRetries(
	ctx context.Context,
	args []string,
	stdIn io.Reader,
	stdOut io.Writer,
	stdErr io.Writer,
	cmdDir string,
	retries int,
	retryDelay int,
	containerName string,
	repeat int,
	ignoreErrors bool) error {
	var aggErrors util.Errors
	for i := 0; i <= repeat; i++ {
		innerErr := pm.RunWithRetries(ctx, args, stdIn, stdOut, stdErr, cmdDir, retries, retryDelay, containerName)
		if innerErr != nil {
			aggErrors = append(aggErrors, innerErr)
		}
	}
	if len(aggErrors) > 0 {
		return errors.New(aggErrors.String())
	}
	return nil
}

// RunWithRetries performs Run with retries.
func (pm *ProcManager) RunWithRetries(
	ctx context.Context,
	args []string,
	stdIn io.Reader,
	stdOut io.Writer,
	stdErr io.Writer,
	cmdDir string,
	retries int,
	retryDelay int,
	containerName string) error {
	attempt := 0
	var err error
	for attempt <= retries {
		// log.Printf("Launching container with name: %s\n", containerName)
		if err = pm.Run(ctx, args, stdIn, stdOut, stdErr, cmdDir); err == nil {
			log.Printf("Successfully executed container: %s\n", containerName)
			break
		} else {
			attempt++
			if attempt <= retries {
				log.Printf("Container failed during run: %s, waiting %d seconds before retrying...\n", containerName, retryDelay)
				time.Sleep(time.Duration(retryDelay) * time.Second)
			} else {
				log.Printf("Container failed during run: %s. No retries remaining.\n", containerName)
			}
		}
	}
	return err
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

	if args == nil {
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
				log.Printf("Failed to kill process. Path: %s, Args: %v, Err: %v\n", cmd.Path, cmd.Args, err)
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

	var errs util.Errors
	for pid, process := range pm.processes {
		if err := process.Kill(); err != nil {
			errs = append(errs, err)
		}
		delete(pm.processes, pid)
	}
	return errs
}
