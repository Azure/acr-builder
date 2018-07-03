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

	"github.com/Azure/acr-builder/baseimages/scanner/util"
	dockerbuild "github.com/docker/cli/cli/command/image/build"
	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/progress"
	"github.com/docker/docker/pkg/streamformatter"
	"github.com/pkg/errors"
)

type dockerSourceType int

// TODO: review and eliminate this.
const (
	dockerSourceUnknown dockerSourceType = iota
	dockerSourceLocal
	dockerSourceGit
	dockerSourceArchive
)

const (
	archiveHeaderSize = 512
)

func (s *Scanner) obtainSourceCode(
	ctx context.Context,
	context string,
	dockerfile string) (workingDir string, sha string, sourceType dockerSourceType, err error) {
	sourceType, workingDir, err = s.getContext(ctx, context, dockerfile)
	if err != nil {
		return workingDir, sha, sourceType, errors.Wrap(err, "failed to obtain source code")
	}

	if sourceType == dockerSourceGit {
		sha, err = s.GetGitCommitID(ctx, workingDir)
	}

	return workingDir, sha, sourceType, err
}

func (s *Scanner) getContext(ctx context.Context, context string, dockerfile string) (dockerSourceType, string, error) {
	isGitURL := util.IsGitURL(context)
	isVstsURL := util.IsVstsGitURL(context)
	isURL := util.IsURL(context)

	// If the context is remote, make the destination folder to clone or untar into.
	if isGitURL || isVstsURL || isURL {
		if _, err := os.Stat(s.destinationFolder); os.IsNotExist(err) {
			// Creates the destination folder if necessary, granting full permissions to the owner.
			if innerErr := os.Mkdir(s.destinationFolder, 0700); innerErr != nil {
				return dockerSourceUnknown, context, innerErr
			}
		}
	}

	if isGitURL || isVstsURL {
		workingDir, err := s.getContextFromGitURL(context)
		return dockerSourceGit, workingDir, err
	} else if isURL {
		sourceType, err := s.getContextFromURL(context)
		return sourceType, s.destinationFolder, err
	}

	return dockerSourceLocal, context, nil
}

func (s *Scanner) getContextFromGitURL(gitURL string) (contextDir string, err error) {
	if _, err := exec.LookPath("git"); err != nil {
		return contextDir, errors.Wrapf(err, "unable to find git")
	}
	contextDir, err = Clone(gitURL, s.destinationFolder)
	if err != nil {
		return contextDir, errors.Wrapf(err, "unable to git clone to a temporary context directory")
	}

	return contextDir, err
}

func (s *Scanner) getContextFromURL(remoteURL string) (sourceType dockerSourceType, err error) {
	response, err := getWithStatusError(remoteURL)
	if err != nil {
		return dockerSourceUnknown, errors.Wrapf(err, "unable to download remote context from %s", remoteURL)
	}

	// TODO: revamp streaming, for now just pipe to buf and discard it.
	var buf bytes.Buffer
	progressOutput := streamformatter.NewProgressOutput(&buf)
	r := progress.NewProgressReader(response.Body, progressOutput, response.ContentLength, "", "Downloading build context")
	defer func(response *http.Response) {
		err := response.Body.Close()
		if err != nil {
			fmt.Printf("Failed to close http response from url: %s, err: %v\n", remoteURL, err)
		}
	}(response)

	err = s.getContextFromReader(r)
	return dockerSourceArchive, err
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
