// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

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
	config          = `{
	"experimental": "enabled",
	"HttpHeaders": {"X-Meta-Source-Client": "ACR-BUILDER"}
}`
)

// dockerLogin performs a docker login
func (b *Builder) dockerLogin(ctx context.Context) error {
	args := []string{
		"docker",
		"run",
		"--name", fmt.Sprintf("acb_docker_login_%s", uuid.New()),
		"--rm",

		// Interactive mode for --password-stdin
		"-i",

		// Mount home
		"--volume", util.GetDockerSock(),
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", "HOME=" + homeWorkDir,

		"docker",
		"login",
		"--username", b.buildOptions.RegistryUsername,
		"--password-stdin",
		b.buildOptions.RegistryName,
	}

	stdIn := strings.NewReader(b.buildOptions.RegistryPassword + "\n")

	var buf bytes.Buffer
	if err := b.cmder.Run(ctx, args, stdIn, &buf, &buf, ""); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to set docker credentials: %s", buf.String()))
	}

	return nil
}

// dockerLoginWithRetries performs a Docker login with retries.
func (b *Builder) dockerLoginWithRetries(ctx context.Context, attempt int) error {
	err := b.dockerLogin(ctx)
	if err != nil {
		if attempt < maxLoginRetries {
			// TODO: exponential backoff
			time.Sleep(500 * time.Millisecond)
			return b.dockerLoginWithRetries(ctx, attempt+1)
		}

		return errors.Wrap(err, "failed to login, ran out of retries")
	}

	return nil
}

// setupConfig initializes ~/.docker/config.json
func (b *Builder) setupConfig(ctx context.Context) error {
	args := []string{
		"docker",
		"run",
		"--name", fmt.Sprintf("acb_init_config_%s", uuid.New()),
		"--rm",

		// Home
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", "HOME=" + homeWorkDir,

		"--volume", util.GetDockerSock(),
		"--entrypoint", "bash",
		"ubuntu",
		"-c", "mkdir -p ~/.docker && cat << EOF > ~/.docker/config.json\n" + config + "\nEOF",
	}

	var buf bytes.Buffer
	if err := b.cmder.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to setup config: %s", buf.String()))
	}

	return nil
}
