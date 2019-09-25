// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/util"
)

func TestCreateBuildTask(t *testing.T) {
	var (
		isolation       = "someIsolation"
		pull            = true
		labels          = []string{"foo=bar"}
		noCache         = true
		dockerfile      = "HelloWorld/Dockerfile"
		tags            = []string{"{{.Run.Registry}}/foo:latest", "bar/qux"}
		buildArgs       = []string{"someArg"}
		secretBuildArgs = []string{"someSecretArg"}
		target          = "someTarget"
		platform        = "somePlatform"
		buildContext    = "src"
		registry        = "foo.azurecr.io"
		push            = true
		creds           = []string{`{"registry":"foo.azurecr.io","username":"user","userNameProviderType":"opaque","password":"pw","passwordProviderType":"opaque"}`}
		workingDir      = ""
	)

	type ctxKey string
	var debug ctxKey = "debug"
	ctx := context.WithValue(context.Background(), debug, true)

	task, err := CreateBuildTask(ctx, &TaskCreateOptions{
		Isolation:        isolation,
		Pull:             pull,
		Labels:           labels,
		NoCache:          noCache,
		Dockerfile:       dockerfile,
		Tags:             tags,
		BuildArgs:        buildArgs,
		SecretBuildArgs:  secretBuildArgs,
		Target:           target,
		Platform:         platform,
		BuildContext:     buildContext,
		Registry:         registry,
		Push:             push,
		Credentials:      creds,
		WorkingDirectory: workingDir,
	})
	if err != nil {
		t.Fatalf("failed to create build task, err: %v", err)
	}
	if len(task.Credentials) == 0 {
		t.Fatalf("Expected to create credentials but no credentials were created")
	}

	taskCreds := task.Credentials[0]
	expectedCreds, err := graph.CreateRegistryCredentialFromString(creds[0])
	if err != nil {
		t.Fatalf("failed to create registry credentials from string, err: %v", err)
	}
	if !taskCreds.Equals(expectedCreds) {
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
	expectedCmd := "--isolation=someIsolation --pull --label foo=bar --no-cache -f HelloWorld/Dockerfile " +
		"-t foo.azurecr.io/foo:latest -t foo.azurecr.io/bar/qux " +
		"--build-arg someArg --build-arg someSecretArg --target someTarget --platform somePlatform src"
	if expectedCmd != buildStep.Build {
		t.Fatalf("expected %s as the build command, but got %s", expectedCmd, buildStep.Build)
	}
}
