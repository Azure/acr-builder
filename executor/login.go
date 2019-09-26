// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package executor

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure/acr-builder/util"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	maxLoginRetries = 3
)

// dockerLogin performs a docker login
func (e *Executor) dockerLogin(ctx context.Context, registry string, user string, pw string) error {
	args := []string{
		"docker",
		"run",
		"--name", fmt.Sprintf("acb_docker_login_%s", uuid.New()),
		"--rm",

		// Interactive mode for --password-stdin
		"-i",

		// Mount home
		"--volume", util.DockerSocketVolumeMapping,
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", homeEnv,

		dockerCLIImageName,
		"login",
		"--username", user,
		"--password-stdin",
		registry,
	}

	stdIn := strings.NewReader(pw + "\n")

	var buf bytes.Buffer
	if err := e.procManager.Run(ctx, args, stdIn, &buf, &buf, ""); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to set docker credentials: %s", buf.String()))
	}

	return nil
}

// dockerLoginWithRetries performs a Docker login with retries.
func (e *Executor) dockerLoginWithRetries(ctx context.Context, registry string, user string, pw string, attempt int) error {
	err := e.dockerLogin(ctx, registry, user, pw)
	if err != nil {
		if attempt < maxLoginRetries {
			time.Sleep(util.GetExponentialBackoff(attempt))
			return e.dockerLoginWithRetries(ctx, registry, user, pw, attempt+1)
		}

		return errors.Wrap(err, "failed to login, ran out of retries")
	}

	return nil
}
