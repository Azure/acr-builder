// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"os"
	"bytes"
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const (
	config = `{"experimental": "enabled", "HttpHeaders": {"X-Meta-Source-Client": "ACR-BUILDER"}}`
)

// setupConfig initializes ~/.docker/config.json
func (b *Builder) setupConfig(ctx context.Context) error {
	imageName := ""
	if imageName = os.Getenv("ACB_CONFIGIMAGENAME"); imageName == "" {
		imageName = configImageName
	}

	args := []string{
		"docker",
		"run",
		"--name", fmt.Sprintf("acb_init_config_%s", uuid.New()),
		"--rm",

		// Home
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", homeEnv,
		"--entrypoint", "powershell",
		imageName,
		"mkdir ~/.docker; Out-File -InputObject '" + config + "' -FilePath ~/.docker/config.json -Encoding ASCII",
	}

	var buf bytes.Buffer
	if err := b.procManager.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to setup config: %s", buf.String()))
	}

	return nil
}
