// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

//go:build linux || darwin

package builder

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	config = `{
		"experimental": "enabled",
		"HttpHeaders": {"X-Meta-Source-Client": "azure/acr/tasks"}
	}`
)

// setupConfig initializes ~/.docker/config.json
func (b *Builder) setupConfig(ctx context.Context) error {
	args := []string{
		"docker",
		"run",
		"--name", fmt.Sprintf("acb_init_config_%s", uuid.New()),
		"--rm",

		// Home
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", homeEnv,
		"--entrypoint", "bash",
		configImageName,
		"-c", "mkdir -p ~/.docker && cat << EOF > ~/.docker/config.json\n" + config + "\nEOF",
	}

	var buf bytes.Buffer
	if err := b.procManager.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to setup config, msg: %s", buf.String()))
	}

	return nil
}
