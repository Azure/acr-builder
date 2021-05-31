package builder

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Azure/acr-builder/graph"
	"github.com/Azure/acr-builder/pkg/image"
	"github.com/containerd/containerd/errdefs"
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
	if ref.Digest != "" {
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
	// Note: (bug), if the final image is NOT pushed in this build,
	// Then the `Resolve` command below will resolve the digest of the LAST pushed image with the same tag (IFF tag is provided)
	// Then, the unwanted behavior of runtime dependencies found in THIS run being tracked on OLD image occurs. 
	// For ex., consider following step:
	// - id: build
	//   build: -t $Registry/acb:latest -f Dockerfile .
	//   env: ["DOCKER_BUILDKIT=1"]
	// 
	// Since we do not push the final image "acb:latest", `Resolve` will get digest of the "acb:latest" IFF it was ever pushed before. 
	// Lets assume old digest was sha256:xyz
	// Then, "$Registry/acb:latest" with digest "sha256:xyz" will start to track dependencies from THIS run.
	// But, I assume MOST tasks always Push after Build, so the dependencies are tracked on the last Pushed digest, so it won't be an issue in MOST cases.
	// TODO: We need to skip attaching the digest to the final image if the final image wasn't pushed in this Task run.
	_, desc, err := resolver.Resolve(ctx, imageRef)
	if err == nil {
		ref.Digest = desc.Digest.String()
		return nil
	}
	// If the final image is not pushed ever, it will not have any digest.
	if errdefs.IsNotFound(err) {
		return nil
	}

	return errors.Wrapf(err, "Failed to Resolve the reference '%s'", ref.Reference)
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