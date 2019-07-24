// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

const (
	// NoBaseImageSpecifierLatest is the empty base image
	// Note that :latest is not valid in the FROM clause, but we're
	// always appending :latest to tags during processing.
	NoBaseImageSpecifierLatest = "scratch:latest"

	// DockerHubRegistry is the docker hub registry
	DockerHubRegistry = "registry.hub.docker.com"

	// homeVol is the volume to manage $HOME
	homeVol = "home"

	scannerImageName   = "acb"
	dockerCLIImageName = "docker"

	configTimeoutInSec  = 60 * 5  // 5 minutes
	loginTimeoutInSec   = 60 * 5  // 5 minutes
	digestsTimeoutInSec = 60 * 5  // 5 minutes
	scrapeTimeoutInSec  = 60 * 15 // 15 minutes

	// build cache constants
	buildkitContainerRunTimeout     = 60 * 2 // 2 minutes
	buildkitContainerInitRetries    = 3
	buildkitContainerInitRetryDelay = 5 // 5 seconds
	buildkitContainerName           = "buildkitcontainer"
	buildkitContainerInitRepeat     = 0 // no repetition for retries
)
