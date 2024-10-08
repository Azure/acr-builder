// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package builder

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

type remoteDigest struct {
	registryCreds graph.RegistryLoginCredentials
}

func newRemoteDigest(creds graph.RegistryLoginCredentials) *remoteDigest {
	return &remoteDigest{
		registryCreds: creds,
	}
}

var _ DigestHelper = &remoteDigest{}

func (d *remoteDigest) PopulateDigest(ctx context.Context, ref *image.Reference) error {
	if ref == nil {
		return nil
	}
	if ref.Digest != "" {
		return nil
	}
	if ref.Reference == NoBaseImageSpecifierLatest {
		return nil
	}

	config := docker.RegistryHost{
		Client: http.DefaultClient,
		Scheme: "https",
		Path:   "/v2",
		// NOTE: it requires both pull and resolve capabilities to get the digest
		Capabilities: docker.HostCapabilityPull | docker.HostCapabilityResolve,
	}

	if cred, ok := d.registryCreds[ref.Registry]; ok {
		if cred.Username.ResolvedValue == "" || cred.Password.ResolvedValue == "" {
			return fmt.Errorf("error fetching credentials for '%s'", ref.Registry)
		}

		config.Authorizer = docker.NewDockerAuthorizer(
			docker.WithAuthCreds(func(hostName string) (string, string, error) {
				if hostName != ref.Registry {
					return "", "", fmt.Errorf("hostName '%s' does not match the registry '%s'", hostName, ref.Registry)
				}
				return cred.Username.ResolvedValue, cred.Password.ResolvedValue, nil
			}),
		)
	} else {
		config.Authorizer = docker.NewDockerAuthorizer(
			docker.WithAuthCreds(func(hostName string) (string, string, error) {
				if hostName != ref.Registry {
					return "", "", fmt.Errorf("hostName '%s' does not match the registry '%s'", hostName, ref.Registry)
				}
				// NOTE: empty credential for anonymous access
				return "", "", nil
			}),
		)
	}
	opts := docker.ResolverOptions{
		Hosts: func(hostName string) ([]docker.RegistryHost, error) {
			config.Host = hostName
			return []docker.RegistryHost{config}, nil
		},
	}
	resolver := docker.NewResolver(opts)
	imageRef, err := getReferencePath(ref)
	if err != nil {
		return err
	}

	_, desc, err := resolver.Resolve(ctx, imageRef)
	if err != nil {
		return errors.Wrapf(err, "Failed to Resolve the reference '%s'", ref.Reference)
	}

	ref.Digest = desc.Digest.String()
	return nil
}

func getReferencePath(ref *image.Reference) (string, error) {
	fullRefPath := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
	tag := "latest"
	if ref.Tag != "" {
		tag = ref.Tag
	}
	fullRefPath = fmt.Sprintf("%s:%s", fullRefPath, tag)
	fullRef, err := reference.Parse(fullRefPath)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse the reference %s", ref.Reference)
	}
	return fullRef.String(), nil
}
