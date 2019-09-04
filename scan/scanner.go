// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"context"
	"path/filepath"

	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/pkg/errors"
)

// Scanner scans Dockerfiles.
type Scanner struct {
	procManager       *procmanager.ProcManager
	context           string
	dockerfile        string
	destinationFolder string
	buildArgs         []string
	tags              []string
	target            string
}

// NewScanner creates a new Scanner.
func NewScanner(pm *procmanager.ProcManager, sourceContext string, dockerfile string, destination string, buildArgs []string, tags []string, target string) (*Scanner, error) {
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
		context:           sourceContext,
		dockerfile:        dockerfile,
		destinationFolder: destination,
		buildArgs:         buildArgs,
		tags:              tags,
		target:            target,
	}, nil
}

// Scan scans a Dockerfile for dependencies.
func (s *Scanner) Scan(ctx context.Context) (deps []*image.Dependencies, err error) {
	workingDir, sha, _, err := s.ObtainSourceCode(ctx, s.context)
	if err != nil {
		return deps, errors.Wrap(err, "failed to download source code")
	}

	deps, err = s.ScanForDependencies(s.context, workingDir, s.dockerfile, s.buildArgs, s.tags, s.target)
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
