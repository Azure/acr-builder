// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"

	"github.com/Azure/acr-builder/util"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"
)

const (
	maxRetries = 3
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
		fmt.Println("Getting context from Git URL")
		workingDir, err := s.getContextFromGitURL(scContext)
		return workingDir, err
	} else if isURL {
		fmt.Printf("Getting context from URL %s\n", scContext)
		err := s.getContextFromURL(scContext)
		return s.destinationFolder, err
	} else if isRegistryArtifact {
		fmt.Println("Getting context from registry")
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
	attempt := 0
	for attempt < maxRetries {
		var response *http.Response
		response, err = getWithStatusError(remoteURL)
		if err != nil {
			return errors.Wrapf(err, "unable to download remote context from %s", remoteURL)
		}

		fmt.Printf("Read context with status code %d\n", response.StatusCode)
		fmt.Printf("Read context of %d bytes\n", response.ContentLength)

		// TODO: revamp output streaming, for now just discard it
		progressOutput := streamformatter.NewProgressOutput(io.Discard)

		var rc io.ReadCloser = progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", "Downloading build context")
		defer rc.Close()

		err = s.getContextFromReader(rc)
		if err == nil {
			return nil
		}

		attempt++
		time.Sleep(util.GetExponentialBackoff(attempt))
	}

	return err
}

func (s *Scanner) getContextFromReader(r io.Reader) (err error) {
	fmt.Println("starting to untar context")
	err = archive.Untar(r, s.destinationFolder, nil)
	if err != nil {
		return errors.Wrap(err, "failed to untar context")
	}

	return err
}

func (s *Scanner) getContextFromRegistry(ctx context.Context, registryArtifact string) (err error) {
	src, err := remote.NewRepository(registryArtifact)
	if err != nil {
		return errors.Wrapf(err, "failed to parse artifact %s", registryArtifact)
	}
	src.Client = &auth.Client{
		Header: http.Header{
			"User-Agent":           {"oras-go"},
			"X-Meta-Source-Client": {"azure/acr/tasks"},
		},
		Cache: auth.DefaultCache,
		Credential: func(ctx context.Context, registry string) (auth.Credential, error) {
			// If no matching credential found, attempt an anonymous pull
			if s.credentials[registry] == nil {
				return auth.EmptyCredential, nil
			}

			return auth.Credential{
				Username: s.credentials[registry].Username.ResolvedValue,
				Password: s.credentials[registry].Password.ResolvedValue,
			}, nil
		},
	}

	dest, err := file.New(s.destinationFolder)
	if err != nil {
		return errors.Wrapf(err, "unable to pull artifact to %s", s.destinationFolder)
	}
	defer dest.Close()

	fmt.Printf("Pulling from %s and saving to %s...\n", registryArtifact, s.destinationFolder)

	desc, err := oras.Copy(ctx, src, src.Reference.Reference, dest, "", oras.DefaultCopyOptions)
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
			time.Sleep(util.GetExponentialBackoff(attempt))
			attempt++
			continue
		}

		if resp.StatusCode < 400 {
			return resp, nil
		}

		msg := fmt.Sprintf("failed to GET %s with status %s", url, resp.Status)

		// If the status code is 4xx then read the body and return an error
		// If the status code is a 5xx then check the body on the final attempt and return an error.
		if resp.StatusCode < 500 || attempt == maxRetries-1 {
			body, bodyReadErr := io.ReadAll(resp.Body)
			resp.Body.Close()

			if bodyReadErr != nil {
				return nil, errors.Wrap(bodyReadErr, fmt.Sprintf("%s: error reading body", msg))
			}
			return nil, errors.Errorf("%s: %s", msg, bytes.TrimSpace(body))
		}

		time.Sleep(util.GetExponentialBackoff(attempt))
		attempt++
	}

	return resp, err
}
