package commands

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	dockerbuild "github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/builder/remotecontext/git"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/pkg/errors"
)

// NewDockerSource creates a new passthrough source
func NewDockerSource(context, dockerfile string) build.Source {
	return &DockerSource{
		context:    context,
		dockerfile: dockerfile,
	}
}

// DockerSource is a source we can pass directly into docker
type DockerSource struct {
	// context is the literal string representing docker build context. It can be a directory, archive url, git url or "-"
	context    string
	dockerfile string
	tracker    *DirectoryTracker
	gitHeadRev string
}

// Obtain obtains the source
func (s *DockerSource) Obtain(runner build.Runner) (err error) {
	runContext := runner.GetContext()
	context := runContext.Expand(s.context)
	dockerfile := runContext.Expand(s.dockerfile)
	if context == constants.FromStdin && dockerfile == constants.FromStdin {
		return fmt.Errorf("invalid argument: can't use stdin for both build context and dockerfile")
	}
	var workingDir string
	workingDir, err = s.ensureContext(runner, context, dockerfile)
	if err != nil {
		return
	}
	if workingDir != "" {
		s.tracker, err = ChdirWithTracking(runner, workingDir)
	}
	return
}

// Return returns the source
func (s *DockerSource) Return(runner build.Runner) error {
	if s.tracker != nil {
		return s.tracker.Return(runner)
	}
	return nil
}

// Export exports the source
func (s *DockerSource) Export() []build.EnvVar {
	return []build.EnvVar{
		{Name: constants.ExportsDockerBuildContext, Value: s.context},
	}
}

// Remark makes a remark to the dependencies
func (s *DockerSource) Remark(runner build.Runner, dependencies *build.ImageDependencies) {
	if len(s.gitHeadRev) > 0 {
		dependencies.Git = &build.GitReference{
			GitHeadRev: s.gitHeadRev,
		}
	}
}

// see docker cli image.runbuild
func (s *DockerSource) ensureContext(runner build.Runner, context, dockerfile string) (workingDir string, err error) {
	if context == "" {
		return "", nil
	} else if context == constants.FromStdin {
		return s.ensureContextFromReader(runner, runner.GetStdin(), workingDir, dockerfile)
	} else if urlutil.IsGitURL(context) {
		workingDir, err := s.ensureContextFromGitURL(context)
		if err != nil {
			return "", err
		}
		sha, queryErr := runner.QueryCmd("git", []string{"rev-parse", "--verify", "HEAD"})
		if queryErr != nil {
			fmt.Fprintf(os.Stderr, "Error querying for git head rev: %s, output will not contain git head revision SHA", queryErr)
		} else {
			s.gitHeadRev = sha
		}
		return workingDir, err
	} else if urlutil.IsURL(context) {
		return s.ensureContextFromURL(runner, os.Stdout, workingDir, context, dockerfile)
	}
	var isDir bool
	isDir, err = runner.GetFileSystem().DoesDirExist(context)
	if err != nil {
		err = errors.Wrapf(err, "Failed to look up context from path %s, note that the context path must be a directory. To use archive as a source, please pipe it in with stdin", context)
		return
	}
	if isDir {
		workingDir = context
		return
	}
	return "", fmt.Errorf("Unable to determine context type for context \"%s\". Dependency scanning will NOT work as expected", context)
}

const archiveHeaderSize = 512

// see dockerbuild.GetContextFromReader
func (s *DockerSource) ensureContextFromReader(runner build.Runner, r io.Reader, workingDir, dockerfile string) (tempDir string, err error) {
	buf := bufio.NewReader(r)
	var magic []byte
	magic, err = buf.Peek(archiveHeaderSize)
	if err != nil && err != io.EOF {
		err = fmt.Errorf("failed to peek context header from STDIN: %v", err)
		return
	}
	var fs = runner.GetFileSystem()
	tempDir, err = fs.CreateTempDir()
	if err != nil {
		return
	}

	if dockerbuild.IsArchive(magic) {
		err = archive.Untar(buf, tempDir, nil)
		return
	}

	// TODO: input stream should be read as dockerfile otherwise, populating it to
	// the location where the build task would pick up
	if dockerfile == "" {
		dockerfile = dockerbuild.DefaultDockerfileName
	} else if dockerfile == "-" {
		dockerfile = dockerbuild.DefaultDockerfileName
		// Following the same undesirable behavior from docker cli in the special case "echo $dockerfile | docker build -f - $docker_file_url"
		fmt.Fprintf(os.Stderr, "Warning: Dockerfile from context stream would be overwritten by stdin")
	}
	err = fs.WriteFile(filepath.Join(tempDir, dockerfile), buf)
	return
}

// see dockerbuild.GetContextFromGitURL
func (s *DockerSource) ensureContextFromGitURL(gitURL string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", errors.Wrapf(err, "unable to find 'git'")
	}
	checkoutRoot, err := git.Clone(gitURL)
	if err != nil {
		return "", errors.Wrapf(err, "unable to 'git clone' to temporary context directory")
	}
	return checkoutRoot, err
}

// see dockerbuild.GetContextFromGitURL
func (s *DockerSource) ensureContextFromURL(runner build.Runner, out io.Writer, workingDir, remoteURL, dockerfile string) (string, error) {
	response, err := s.getWithStatusError(remoteURL)
	if err != nil {
		return "", errors.Errorf("unable to download remote context %s: %v", remoteURL, err)
	}
	progressOutput := streamformatter.NewProgressOutput(out)
	progReader := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", fmt.Sprintf("Downloading build context from remote url: %s", remoteURL))
	defer func(response *http.Response) {
		err := response.Body.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to close http response from url: %s, error: %s", remoteURL, err)
		}
	}(response)
	return s.ensureContextFromReader(runner, progReader, workingDir, dockerfile)
}

func (s *DockerSource) getWithStatusError(url string) (resp *http.Response, err error) {
	if resp, err = http.Get(url); err != nil {
		return nil, err
	}
	if resp.StatusCode < 400 {
		return resp, nil
	}
	msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)
	body, err := ioutil.ReadAll(resp.Body)
	defer func(resp *http.Response) {
		if err := resp.Body.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing response body: %s", err)
		}
	}(resp)
	if err != nil {
		return nil, errors.Wrapf(err, msg+": error reading body")
	}
	return nil, fmt.Errorf(msg+": %s", bytes.TrimSpace(body))
}
