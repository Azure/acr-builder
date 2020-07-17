// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

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
	"time"

	"github.com/Azure/acr-builder/util"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/deislabs/oras/pkg/content"
	"github.com/deislabs/oras/pkg/oras"
	dockerbuild "github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"
)

const (
	archiveHeaderSize = 512
	maxRetries        = 3
)

// ObtainSourceCode obtains the source code from the specified context.
func (s *Scanner) ObtainSourceCode(ctx context.Context, scContext string) (workingDir string, sha string, branch string, err error) {
	workingDir, err = s.getContext(ctx, scContext)
	if err != nil {
		return workingDir, sha, branch, err
	}

	// it might not be a GitRepo but we still query for CommitID/Branch
	// in case if it errors out, `sha` and `branch` will be "", and we eat the errors
	sha, _ = s.GetGitCommitID(ctx, workingDir)
	branch, _ = s.GetGitBranchName(ctx, workingDir)
	return workingDir, sha, branch, nil
}

func (s *Scanner) getContext(ctx context.Context, scContext string) (workingDir string, err error) {
	isSourceControlURL := util.IsSourceControlURL(scContext)
	isURL := util.IsURL(scContext)
	isRegistryArtifact := util.IsRegistryArtifact(scContext)

	// If the context is remote, make the destination folder to clone or untar into.
	if isSourceControlURL || isURL || isRegistryArtifact {
		if _, err := os.Stat(s.destinationFolder); os.IsNotExist(err) {
			// Creates the destination folder if necessary, granting full permissions to the owner.
			if innerErr := os.Mkdir(s.destinationFolder, 0700); innerErr != nil {
				return scContext, innerErr
			}
		}
	}

	if isSourceControlURL {
		workingDir, err := s.getContextFromGitURL(scContext)
		return workingDir, err
	} else if isURL {
		err := s.getContextFromURL(scContext)
		return s.destinationFolder, err
	} else if isRegistryArtifact {
		err := s.getContextFromRegistry(ctx, util.TrimArtifactPrefix(scContext))
		return s.destinationFolder, err
	}

	return scContext, nil
}

func (s *Scanner) getContextFromGitURL(gitURL string) (contextDir string, err error) {
	if _, err = exec.LookPath("git"); err != nil {
		return contextDir, errors.Wrap(err, "unable to find git")
	}
	contextDir, err = Clone(gitURL, s.destinationFolder)
	if err != nil {
		return contextDir, errors.Wrapf(err, "unable to git clone to %s", s.destinationFolder)
	}
	return contextDir, err
}

func (s *Scanner) getContextFromURL(remoteURL string) (err error) {
	response, err := getWithStatusError(remoteURL)
	if err != nil {
		return errors.Wrapf(err, "unable to download remote context from %s", remoteURL)
	}

	// TODO: revamp streaming, for now just pipe to buf and discard it.
	var buf bytes.Buffer
	progressOutput := streamformatter.NewProgressOutput(&buf)
	r := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", "Downloading build context")
	defer func(response *http.Response) {
		_ = response.Body.Close()
	}(response)

	err = s.getContextFromReader(r)
	return err
}

func (s *Scanner) getContextFromReader(r io.Reader) (err error) {
	buf := bufio.NewReader(r)
	var magic []byte

	// note: (sam) read 2048 magic bytes to accomodate Extended Pax headers.
	magic, err = buf.Peek(archiveHeaderSize * 4)
	if err != nil && err != io.EOF {
		return errors.Wrap(err, "failed to peek context header")
	}

	if dockerbuild.IsArchive(magic) {
		err = archive.Untar(buf, s.destinationFolder, nil)
	}

	return err
}

func (s *Scanner) getContextFromRegistry(ctx context.Context, registryArtifact string) (err error) {
	resolver := docker.NewResolver(docker.ResolverOptions{
		PlainHTTP: false,
		Client:    http.DefaultClient,
		Credentials: func(hostName string) (string, string, error) {
			// If no matching credential found, attempt an anonymous pull
			if s.credentials[hostName] == nil {
				return "", "", nil
			}
			return s.credentials[hostName].Username.ResolvedValue, s.credentials[hostName].Password.ResolvedValue, nil
		},
	})
	fileStore := content.NewFileStore(s.destinationFolder)
	defer fileStore.Close()
	fmt.Printf("Pulling from %s and saving to %s...\n", registryArtifact, s.destinationFolder)
	desc, _, err := oras.Pull(ctx, resolver, registryArtifact, fileStore)
	if err != nil {
		return errors.Wrap(err, "failed to pull artifact from registry")
	}
	fmt.Printf("Pulled from %s with digest %s\n", registryArtifact, desc.Digest)
	return nil
}

// getWithStatusError does an http.Get() and returns an error if the
// status code is 4xx or 5xx.
// It retries if either:
// - There was an error making GET request OR
// - The response is 5xx
func getWithStatusError(url string) (resp *http.Response, err error) {
	attempt := 0

	for attempt < maxRetries {
		resp, err = http.Get(url)
		if err != nil {
			continue
		}
		if resp.StatusCode < 400 {
			return resp, nil
		}

		msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)

		// If the status code is 4xx then read the body and return an error
		// If the status code is a 5xx then check the body on the final attempt and return an error.
		if resp.StatusCode < 500 || attempt == maxRetries-1 {
			body, bodyReadErr := ioutil.ReadAll(resp.Body)
			if bodyReadErr != nil {
				return nil, errors.Wrap(bodyReadErr, fmt.Sprintf("%s: error reading body", msg))
			}

			resp.Body.Close()
			return nil, errors.Errorf("%s: %s", msg, bytes.TrimSpace(body))
		}

		time.Sleep(util.GetExponentialBackoff(attempt))
		attempt++
	}

	return resp, err
}
