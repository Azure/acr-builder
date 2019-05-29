// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
)

func TestGetImageDependencies(t *testing.T) {
	json := `[{"image":{"registry":"registry.hub.docker.com","repository":"library/scanner","tag":"latest","digest":"","reference":"scanner:latest"},` +
		`"runtime-dependency":{"registry":"registry.hub.docker.com","repository":"library/alpine","tag":"latest","digest":"","reference":"alpine:latest"},` +
		`"buildtime-dependency":[{"registry":"registry.hub.docker.com","repository":"library/golang","tag":"1.10-alpine","digest":"test","reference":"golang:1.10-alpine"}],` +
		`"git":{"git-head-revision":"abcdef"}}]`

	deps, err := getImageDependencies(json)
	if err != nil {
		t.Fatalf("Unexpected err while parsing image deps: %v", err)
	}
	if len(deps) != 1 {
		t.Fatalf("Expected 1 dependency, got %d", len(deps))
	}
	r := deps[0]
	if r.Image == nil {
		t.Fatal("Image shouldn't be nil")
	}
	if r.Runtime == nil {
		t.Fatal("Runtime shouldn't be nil")
	}
	if len(r.Buildtime) != 1 {
		t.Fatalf("Expected 1 buildtime dep, got %d", len(r.Buildtime))
	}
	if r.Git == nil {
		t.Fatal("git shouldn't be nil")
	}

	expectedImage := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/scanner",
		Tag:        "latest",
		Digest:     "",
		Reference:  "scanner:latest",
	}
	expectedRuntimeDep := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/alpine",
		Tag:        "latest",
		Digest:     "",
		Reference:  "alpine:latest",
	}
	expectedBuildDep := &image.Reference{
		Registry:   "registry.hub.docker.com",
		Repository: "library/golang",
		Tag:        "1.10-alpine",
		Digest:     "test",
		Reference:  "golang:1.10-alpine",
	}
	expectedGitHeadRev := "abcdef"

	if !image.Equals(r.Image, expectedImage) {
		t.Errorf("Invalid image ref. Expected %s, got %s", expectedImage.String(), r.Image.String())
	}
	if !image.Equals(r.Runtime, expectedRuntimeDep) {
		t.Errorf("Invalid runtime dep. Expected %s, got %s", expectedRuntimeDep.String(), r.Runtime.String())
	}
	if !image.Equals(r.Buildtime[0], expectedBuildDep) {
		t.Errorf("Invalid buildtime dep. Expected %s, got %s", expectedBuildDep.String(), r.Buildtime[0].String())
	}
	if r.Git.GitHeadRev != expectedGitHeadRev {
		t.Errorf("Invalid git head rev. Expected %s, got %s", expectedGitHeadRev, r.Git.GitHeadRev)
	}
}

func TestGetBuildDockerRunArgs(t *testing.T) {
	builder := &Builder{}
	actualCmds := builder.getDockerRunArgs("volName", "stepWorkDir", &graph.Step{ID: "id", Build: "-f Dockerfile ."}, []string{"foo=bar", "HOME=qux"}, "", "docker build -f Dockerfile .")

	var expectedCmds []string

	if runtime.GOOS == windowsOS {
		expectedCmds = []string{
			"powershell.exe",
			"-Command",
			"docker run --rm --name id --volume volName:c:\\workspace --volume \\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine --volume home:c:\\acb\\home --env USERPROFILE=c:\\acb\\home --env foo=bar --env HOME=qux --workdir c:\\workspace/stepWorkDir docker build -f Dockerfile .",
		}
	} else {
		expectedCmds = []string{
			"/bin/sh",
			"-c",
			"docker run --rm --name id --volume volName:/workspace --volume /var/run/docker.sock:/var/run/docker.sock --volume home:/acb/home --env HOME=/acb/home --env foo=bar --env HOME=qux --workdir /workspace/stepWorkDir docker build -f Dockerfile .",
		}
	}

	if !reflect.DeepEqual(actualCmds, expectedCmds) {
		t.Errorf("invalid docker run args, expected %v but got %v", expectedCmds, actualCmds)
	}
}

func TestGetNonBuildDockerRunArgs(t *testing.T) {
	builder := &Builder{}
	actualCmds := builder.getDockerRunArgs("volName", "stepWorkDir", &graph.Step{ID: "id"}, []string{"foo=bar"}, "", "hello-world")

	var expectedCmds []string

	if runtime.GOOS == windowsOS {
		expectedCmds = []string{
			"powershell.exe",
			"-Command",
			"docker run --rm --isolation hyperv --name id --volume volName:c:\\workspace --volume \\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine --volume home:c:\\acb\\home --env USERPROFILE=c:\\acb\\home --env foo=bar --workdir c:\\workspace/stepWorkDir hello-world",
		}
	} else {
		expectedCmds = []string{
			"/bin/sh",
			"-c",
			"docker run --rm --name id --volume volName:/workspace --volume /var/run/docker.sock:/var/run/docker.sock --volume home:/acb/home --env HOME=/acb/home --env foo=bar --workdir /workspace/stepWorkDir hello-world",
		}
	}

	if !reflect.DeepEqual(actualCmds, expectedCmds) {
		t.Errorf("invalid docker run args, expected %v but got %v", expectedCmds, actualCmds)
	}
}

func TestGetScanArgs(t *testing.T) {
	tests := []struct {
		containerName         string
		volName               string
		containerWorkspaceDir string
		stepWorkDir           string
		dockerfile            string
		outputDir             string
		tags                  []string
		buildArgs             []string
		context               string
		expected              string
	}{
		{
			"containerName",
			"volumeName",
			"workspaceDir",
			"workingDirectory",
			"Dockerfile",
			"OutputDirectory",
			[]string{"tag1", "tag2"},
			[]string{"arg1=a", "arg2=b"},
			"someContext",
			"docker run --rm " +
				"--name containerName " +
				"--volume volumeName" + ":workspaceDir " +
				"--workdir " + normalizeWorkDir("workingDirectory") + " " +
				"--volume " + homeVol + ":" + homeWorkDir + " " +
				"--env " + homeEnv + " " +
				"acb scan -f Dockerfile --destination OutputDirectory " +
				"-t tag1 -t tag2 --build-arg arg1=a --build-arg arg2=b someContext",
		},
	}

	for _, test := range tests {
		actual := strings.Join(
			getScanArgs(
				test.containerName,
				test.volName,
				test.containerWorkspaceDir,
				test.stepWorkDir,
				test.dockerfile,
				test.outputDir,
				test.tags,
				test.buildArgs,
				test.context),
			" ")
		if test.expected != actual {
			t.Fatalf("Expected\n%s\nbut got\n%s", test.expected, actual)
		}
	}
}
