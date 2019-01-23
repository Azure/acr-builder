// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"context"
	"path/filepath"

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
}

// NewScanner creates a new Scanner.
func NewScanner(pm *procmanager.ProcManager, context string, dockerfile string, destination string, buildArgs []string, tags []string) (*Scanner, error) {
	// NOTE (bindu): vendor/github.com/docker/docker/pkg/idtools/idtools_unix.go#mkdirAs (L51-60) looks for "/" to determine the root folder.
	// But if it is a relative path, the code will enter dead-loop. Ensure passing in the absolute path to workaround the bug.
	var err error
	if !filepath.IsAbs(destination) {
		if destination, err = filepath.Abs(destination); err != nil {
			return nil, err
		}
	}

	return &Scanner{
		procManager:       pm,
		context:           context,
		dockerfile:        dockerfile,
		destinationFolder: destination,
		buildArgs:         buildArgs,
		tags:              tags,
	}, nil
}

// Scan scans a Dockerfile for dependencies.
func (s *Scanner) Scan(ctx context.Context) (deps []*image.Dependencies, err error) {
	workingDir, sha, err := s.ObtainSourceCode(ctx, s.context)
	if err != nil {
		return deps, err
	}

	deps, err = s.ScanForDependencies(s.context, workingDir, s.dockerfile, s.buildArgs, s.tags)
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
