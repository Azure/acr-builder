// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/vaults"
	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
)

const (
	maxTimeout time.Duration = 1<<63 - 1
	// DefaultSecretResolveTimeout is the default timeout for resolving a secret which is 2 minute
	DefaultSecretResolveTimeout time.Duration = time.Minute * 2
)

type secretResolveChannel struct {
	resolvedChan chan graph.SecretValue
	timeoutChan  func() <-chan struct{}
}

// ResolveSecretFunc is a function that resolves the secret to its value and sends through the ResolvedChan of the secret. Any errors during resolve are send through errorChan
type ResolveSecretFunc func(ctx context.Context, azureVaultResourceURL string, secret *graph.Secret, errorChan chan error)

// SecretResolver defines how a secret is resolved.
type SecretResolver struct {
	Resolve          ResolveSecretFunc
	resolveTimeout   time.Duration
	azureEnvironment azure.Environment
}

// NewSecretResolver creates a resolver with the given resolve function.
func NewSecretResolver(resolveFunc ResolveSecretFunc, azureEnvironmentName string, resolveTimeout time.Duration) (*SecretResolver, error) {
	if resolveFunc == nil {
		resolveFunc = resolveSecret
	}
	if azureEnvironmentName == "" {
		azureEnvironmentName = azure.PublicCloud.Name
	}

	azureEnvironment, err := azure.EnvironmentFromName(azureEnvironmentName)
	if err != nil {
		return nil, errors.Wrap(err, "invalid azure environment name")
	}

	return &SecretResolver{Resolve: resolveFunc, azureEnvironment: azureEnvironment, resolveTimeout: resolveTimeout}, nil
}

// ResolveSecrets returns a list of resolved secrets and returns error if there is any failure in resolving a secret.
func (secretResolver *SecretResolver) ResolveSecrets(ctx context.Context, secrets []*graph.Secret) (Values, error) {

	resolvedSecrets := Values{}

	if len(secrets) == 0 {
		return resolvedSecrets, nil
	}

	// We will resolve in batches of 5 to avoid throttling errors on the vault providers
	batchSize := 5
	errorChan := make(chan error)
	for index := 0; index < len(secrets); index += batchSize {
		endIndex := index + batchSize

		if endIndex > len(secrets) {
			endIndex = len(secrets)
		}

		var secretChannels []secretResolveChannel

		for _, secret := range secrets[index:endIndex] {
			if secret == nil {
				continue
			}

			if secret.ResolvedChan == nil {
				secret.ResolvedChan = make(chan graph.SecretValue)
			}
			ctxWithTimeout, cancel := context.WithTimeout(ctx, secretResolver.resolveTimeout)
			defer cancel()
			secretChannels = append(secretChannels, secretResolveChannel{secret.ResolvedChan, ctxWithTimeout.Done})
			go secretResolver.Resolve(ctxWithTimeout, secretResolver.azureEnvironment.KeyVaultEndpoint, secret, errorChan)
		}

		// Block until either:
		// - timeout in fetching any of the secrets.
		// - The global context expires
		// - Resolving a secret has error
		// - All secrets are resolved successfully
		for _, ch := range secretChannels {
			select {
			case <-ch.timeoutChan():
				return resolvedSecrets, errors.New("timeout in fetching secrets")
			case <-ctx.Done():
				return resolvedSecrets, ctx.Err()
			case secretValue := <-ch.resolvedChan:
				resolvedSecrets[secretValue.ID] = secretValue.Value
			case err := <-errorChan:
				return resolvedSecrets, err
			}
		}
	}

	return resolvedSecrets, nil
}

func resolveSecret(ctx context.Context, azureVaultResourceURL string, secret *graph.Secret, errorChan chan error) {
	if secret == nil {
		errorChan <- errors.New("secret cannot be nil")
		return
	}

	if secret.IsAkvSecret() {
		secretConfig, err := vaults.NewAKVSecretConfig(secret.Akv, secret.MsiClientID, azureVaultResourceURL)
		if err != nil {
			errorChan <- errors.Wrap(err, "failed to create azure keyvault secret config")
			return
		}

		secretValue, err := secretConfig.GetValue(ctx)
		if err != nil {
			errorChan <- errors.Wrap(err, "failed to fetch azure key vault secret")
			return
		}

		secret.ResolvedChan <- graph.SecretValue{ID: secret.ID, Value: secretValue}
		return
	}

	errorChan <- fmt.Errorf("cannot resolve secret with ID: %s", secret.ID)
}
