// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package build

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/templating"
	"github.com/Azure/acr-builder/util"
)

func TestCreateBuildTask(t *testing.T) {
	var (
		isolation       = "someIsolation"
		pull            = true
		labels          = []string{}
		noCache         = false
		dockerfile      = "HelloWorld/Dockerfile"
		tags            = []string{"{{.Run.Registry}}/foo:latest", "bar/qux"}
		buildArgs       = []string{"someArg"}
		secretBuildArgs = []string{"someSecretArg"}
		target          = "someTarget"
		platform        = "somePlatform"
		buildContext    = "src"
		registry        = "foo.azurecr.io"
		renderOpts      = &templating.BaseRenderOptions{
			Registry: registry,
		}
		debug = false
		push  = true
		creds = []string{"foo.azurecr.io;user;pw"}
	)

	task, err := createBuildTask(
		context.Background(),
		isolation,
		pull,
		labels,
		noCache,
		dockerfile,
		tags,
		buildArgs,
		secretBuildArgs,
		target,
		platform,
		buildContext,
		renderOpts,
		debug,
		registry,
		push,
		creds)
	if err != nil {
		t.Fatalf("failed to create build task, err: %v", err)
	}
	if len(task.Credentials) == 0 {
		t.Fatalf("Expected to create credentials but no credentials were created")
	}

	taskCreds := task.Credentials[0]
	expectedCreds, _ := graph.CreateCredentialFromString(creds[0])
	if taskCreds.RegistryName != expectedCreds.RegistryName ||
		taskCreds.RegistryUsername != expectedCreds.RegistryUsername ||
		taskCreds.RegistryPassword != expectedCreds.RegistryPassword {
		t.Fatalf("expected %v Creds, got %v", expectedCreds, taskCreds)
	}

	numSteps := len(task.Steps)
	expectedSteps := 2
	if numSteps != expectedSteps {
		t.Fatalf("expected %d steps, got %d", expectedSteps, numSteps)
	}

	// When registry information is provided, the resulting tags will be
	// prefixed with the fully qualified registry's name.
	buildStep := task.Steps[0]
	expectedTags := []string{"foo.azurecr.io/foo:latest", "foo.azurecr.io/bar/qux"}
	if !util.StringSequenceEquals(buildStep.Tags, expectedTags) {
		t.Fatalf("expected %v to be the task's tags but got %v", expectedTags, buildStep.Tags)
	}
	expectedCmd := "--isolation=someIsolation --pull -f HelloWorld/Dockerfile " +
		"-t foo.azurecr.io/foo:latest -t foo.azurecr.io/bar/qux " +
		"--build-arg someArg --build-arg someSecretArg --target someTarget --platform somePlatform src"
	if expectedCmd != buildStep.Build {
		t.Fatalf("expected %s as the build command, but got %s", expectedCmd, buildStep.Build)
	}
}
