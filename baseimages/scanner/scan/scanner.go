// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"context"

	"github.com/Azure/acr-builder/baseimages/scanner/models"
	"github.com/Azure/acr-builder/taskmanager"
)

// Scanner scans Dockerfiles.
type Scanner struct {
	taskManager       *taskmanager.TaskManager
	context           string
	dockerfile        string
	destinationFolder string
	buildArgs         []string
	tags              []string
	debug             bool
}

// NewScanner creates a new Scanner.
func NewScanner(tm *taskmanager.TaskManager, context string, dockerfile string, destination string, buildArgs []string, tags []string, debug bool) *Scanner {
	return &Scanner{
		taskManager:       tm,
		context:           context,
		dockerfile:        dockerfile,
		destinationFolder: destination,
		buildArgs:         buildArgs,
		tags:              tags,
		debug:             debug,
	}
}

// Scan scans a Dockerfile for dependencies.
func (s *Scanner) Scan(ctx context.Context) (deps []*models.ImageDependencies, err error) {
	workingDir, sha, _, err := s.obtainSourceCode(ctx, s.context, s.dockerfile)
	if err != nil {
		return deps, err
	}

	deps, err = s.ScanForDependencies(workingDir, s.dockerfile, s.buildArgs, s.tags)
	if err != nil {
		return deps, err
	}

	for _, dep := range deps {
		dep.Git = &models.GitReference{
			GitHeadRev: sha,
		}
	}

	return deps, err
}
