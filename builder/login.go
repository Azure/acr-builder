package builder

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"
)

// dockerLoginWithRetries performs a Docker login with retries.
func (b *Builder) dockerLoginWithRetries(ctx context.Context, attempt int) error {
	err := b.dockerLogin(ctx)
	if err != nil {
		if attempt < maxLoginRetries {
			// TODO: backoffs
			time.Sleep(500 * time.Millisecond)
			return b.dockerLoginWithRetries(ctx, attempt+1)
		}

		fmt.Println("Failed to login; ran out of retries")
		return err
	}

	return nil
}

// dockerLogin performs a Docker login.
func (b *Builder) dockerLogin(ctx context.Context) error {
	bo := b.buildOptions
	args := []string{"docker", "login", "-u", bo.RegistryUsername, "--password-stdin", bo.RegistryName}

	stdIn := strings.NewReader(bo.RegistryPassword + "\n")
	err := b.cmder.Run(ctx, args, stdIn, os.Stdout, os.Stderr, "")
	return err
}
