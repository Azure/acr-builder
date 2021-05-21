package builder

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/containerd/containerd/remotes/docker"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
)

type remoteDigest struct {
	registryCreds graph.RegistryLoginCredentials
}

func NewRemoteDigest(creds graph.RegistryLoginCredentials) *remoteDigest {
	return &remoteDigest{
		registryCreds: creds,
	}
}

var _ DigestHelper = &remoteDigest{}

func (d *remoteDigest) PopulateDigest(ctx context.Context, ref *image.Reference) error {
	if ref == nil {
		return nil
	}
	client := http.DefaultClient
	opts := docker.ResolverOptions{
		Client: client,
	}
	if cred, ok := d.registryCreds[ref.Registry]; ok {
		if cred.Username.ResolvedValue == "" || cred.Password.ResolvedValue == "" {
			return fmt.Errorf("Error fetching credentials for '%s'", ref.Registry)
		}
		// Adds credential resolver if private registry
		opts.Credentials = func(hostName string) (string, string, error) {
			return cred.Username.ResolvedValue, cred.Password.ResolvedValue, nil
		}
	}

	resolver := docker.NewResolver(opts)
	imageRef, err := getReferencePath(ref)
	if err != nil {
		return err
	}
	_, desc, err := resolver.Resolve(ctx, imageRef)

	// If the image is not pushed yet, it will not have any digest.
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "not found") {
		return errors.Wrapf(err, "Failed to Resolve the reference '%s'", ref.Reference)
	}
	ref.Digest = desc.Digest.String()

	return nil
}

func getReferencePath(ref *image.Reference) (string, error) {
	fullRefPath := fmt.Sprintf("%s/%s", ref.Registry, ref.Repository)
	if ref.Tag != "" {
		fullRefPath = fmt.Sprintf("%s:%s", fullRefPath, ref.Tag)
	}
	fullRef, err := reference.Parse(fullRefPath)
	if err != nil {
		return "", errors.Wrapf(err, "Failed to parse the reference %s", ref.Reference)
	}
	return fullRef.String(), nil
}
