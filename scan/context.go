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

	"github.com/Azure/acr-builder/util"
	dockerbuild "github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"
)

const (
	archiveHeaderSize = 512
)

// ObtainSourceCode obtains the source code from the specified context.
func (s *Scanner) ObtainSourceCode(ctx context.Context, context string) (workingDir string, sha string, err error) {
	isGitURL, workingDir, err := s.getContext(ctx, context)
	if err != nil {
		return workingDir, sha, err
	}

	if isGitURL {
		sha, err = s.GetGitCommitID(ctx, workingDir)
	}

	return workingDir, sha, err
}

func (s *Scanner) getContext(ctx context.Context, context string) (bool, string, error) {
	isSourceControlURL := util.IsSourceControlURL(context)
	isURL := util.IsURL(context)

	// If the context is remote, make the destination folder to clone or untar into.
	if isSourceControlURL || isURL {
		if _, err := os.Stat(s.destinationFolder); os.IsNotExist(err) {
			// Creates the destination folder if necessary, granting full permissions to the owner.
			if innerErr := os.Mkdir(s.destinationFolder, 0700); innerErr != nil {
				return false, context, innerErr
			}
		}
	}

	if isSourceControlURL {
		workingDir, err := s.getContextFromGitURL(context)
		return true, workingDir, err
	} else if isURL {
		err := s.getContextFromURL(context)
		return false, s.destinationFolder, err
	}

	return false, context, nil
}

func (s *Scanner) getContextFromGitURL(gitURL string) (contextDir string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return contextDir, errors.Wrapf(err, "unable to find git")
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

	magic, err = buf.Peek(archiveHeaderSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to peek context header from STDIN: %v", err)
	}

	if dockerbuild.IsArchive(magic) {
		err = archive.Untar(buf, s.destinationFolder, nil)
	}

	return err
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
