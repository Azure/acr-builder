package builder

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"

	"github.com/Azure/acr-builder/util"
	dockerbuild "github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/remotecontext/git"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

type dockerSourceType int

const (
	dockerSourceUnknown dockerSourceType = iota
	dockerSourceLocal
	dockerSourceGit
	dockerSourceDockerfile
	dockerSourceArchive
)

const (
	archiveHeaderSize = 512
	defaultBlankImage = "nothing"
)

func (b *Builder) obtainSourceCode(
	ctx context.Context,
	context string,
	dockerfile string) (workingDir string, sha string, err error) {
	sourceType, workingDir, err := b.getContext(ctx, context, dockerfile)
	if err != nil {
		return workingDir, sha, errors.Wrap(err, "failed to obtain source code")
	}

	if sourceType == dockerSourceGit {
		sha, err = b.GetGitCommitID(ctx, workingDir)
	}

	return workingDir, sha, err
}

func (b *Builder) getContext(
	ctx context.Context,
	context string,
	dockerfile string) (sourceType dockerSourceType, workingDir string, err error) {

	if urlutil.IsGitURL(context) || util.IsVstsGitURL(context) {
		workingDir, err := b.getContextFromGitURL(context)
		return dockerSourceGit, workingDir, err
	} else if urlutil.IsURL(context) {
		return b.getContextFromURL(context)
	}

	return dockerSourceLocal, context, err
}

func (b *Builder) getContextFromURL(remoteURL string) (sourceType dockerSourceType, workingDir string, err error) {
	response, err := getWithStatusError(remoteURL)
	if err != nil {
		return dockerSourceUnknown, workingDir, errors.Wrapf(err, "unable to download remote context from %s", remoteURL)
	}
	progressOutput := streamformatter.NewProgressOutput(os.Stdout)
	r := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", "Downloading build context")
	defer func(response *http.Response) {
		err := response.Body.Close()
		if err != nil {
			fmt.Printf("Failed to close http response from url: %s, err: %v\n", remoteURL, err)
		}
	}(response)

	workingDir, err = b.getContextFromReader(r)
	return dockerSourceArchive, workingDir, err
}

func (b *Builder) getContextFromGitURL(gitURL string) (contextDir string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return contextDir, errors.Wrapf(err, "unable to find git")
	}
	contextDir, err = git.Clone(gitURL)
	if err != nil {
		return contextDir, errors.Wrapf(err, "unable to git clone to a temporary context directory")
	}

	return contextDir, err
}

func (b *Builder) getContextFromReader(r io.Reader) (workingDir string, err error) {
	buf := bufio.NewReader(r)
	var magic []byte

	magic, err = buf.Peek(archiveHeaderSize)
	if err != nil && err != io.EOF {
		return workingDir, fmt.Errorf("failed to peek context header from STDIN: %v", err)
	}
	workingDir, err = ioutil.TempDir("", "build-context-")
	if err != nil {
		return workingDir, errors.Errorf("unable to create temp context folder: %v", err)
	}

	if dockerbuild.IsArchive(magic) {
		err = archive.Untar(buf, workingDir, nil)
	}

	return workingDir, err
}

// getDockerRunArgs populates the args for running a Docker container.
func (b *Builder) getDockerRunArgs(stepID string, stepWorkDir string) []string {
	args := []string{"docker", "run"}

	if rmContainer {
		args = append(args, "--rm")
	}

	args = append(args,
		"--name", fmt.Sprintf("rally_step_%s", stepID),
		"--volume", "/var/run/docker.sock:/var/run/docker.sock",
		"--volume", b.workspaceDir+":"+containerWorkspaceDir,
		"--workdir", normalizeWorkDir(stepWorkDir),

		"--volume", rallyHomeVol+":"+rallyHomeWorkDir,

		// Set $HOME to the home volume.
		"--env", "HOME="+rallyHomeVol,
		"--privileged",
	)
	return args
}

func (b *Builder) copyContext(ctx context.Context, workingDir string, mountLocation string) error {
	containerName := fmt.Sprintf("rally_context_share_%s", uuid.New())
	cArgs := []string{
		"docker",
		"create",
		"--name", containerName,
		"--volume", b.workspaceDir + ":" + containerWorkspaceDir,

		"--volume", rallyHomeVol + ":" + rallyHomeWorkDir,
		// Set $HOME to the home volume.
		"--env", "HOME=" + rallyHomeVol,
		defaultBlankImage,
	}

	err := b.cmder.Run(ctx, cArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to create container %s to share context", containerName)
	}

	cpArgs := []string{
		"docker",
		"cp",
		workingDir,
		fmt.Sprintf("%s:%s", containerName, normalizeWorkDir(mountLocation)),
	}

	err = b.cmder.Run(ctx, cpArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to copy context to container %s", containerName)
	}

	rmArgs := []string{
		"docker",
		"rm",
		containerName,
	}

	err = b.cmder.Run(ctx, rmArgs, nil, os.Stdout, os.Stderr, "")
	if err != nil {
		return errors.Wrapf(err, "Failed to clean up shared context container %s", containerName)
	}

	return nil
}

// getWithStatusError does an http.Get() and returns an error if the
// status code is 4xx or 5xx.
func getWithStatusError(url string) (resp *http.Response, err error) {
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	if resp.StatusCode < 400 {
		return resp, nil
	}
	msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrapf(err, msg+": error reading body")
	}

	_ = resp.Body.Close()
	return nil, errors.Errorf(msg+": %s", bytes.TrimSpace(body))
}

// normalizeWorkDir normalizes a step's working directory.
func normalizeWorkDir(stepWorkDir string) string {
	// If the step's directory is absolute, use it instead of /workspace/...
	if path.IsAbs(stepWorkDir) {
		return path.Clean(stepWorkDir)
	}

	return path.Clean(path.Join("/workspace", stepWorkDir))
}
