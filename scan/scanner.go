// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"context"

	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
)

// Scanner scans Dockerfiles.
type Scanner struct {
	procManager       *procmanager.ProcManager
	context           string
	dockerfile        string
	destinationFolder string
	buildArgs         []string
	tags              []string
	debug             bool
}

// NewScanner creates a new Scanner.
func NewScanner(pm *procmanager.ProcManager, context string, dockerfile string, destination string, buildArgs []string, tags []string, debug bool) *Scanner {
	return &Scanner{
		procManager:       pm,
		context:           context,
		dockerfile:        dockerfile,
		destinationFolder: destination,
		buildArgs:         buildArgs,
		tags:              tags,
		debug:             debug,
	}
}

// Scan scans a Dockerfile for dependencies.
func (s *Scanner) Scan(ctx context.Context) (deps []*image.Dependencies, err error) {
	workingDir, sha, _, err := s.ObtainSourceCode(ctx, s.context)
	if err != nil {
		return deps, err
	}

	deps, err = s.ScanForDependencies(workingDir, s.dockerfile, s.buildArgs, s.tags)
	if err != nil {
		return deps, err
	}

	for _, dep := range deps {
		dep.Git = &image.GitReference{
			GitHeadRev: sha,
		}
	}

	return deps, err
}
