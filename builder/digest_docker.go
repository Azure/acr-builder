package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/Azure/acr-builder/pkg/image"
	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

type dockerStoreDigest struct {
	procManager *procmanager.ProcManager
	debug       bool
}

func NewDockerStoreDigest(procManager *procmanager.ProcManager, debug bool) *dockerStoreDigest {
	return &dockerStoreDigest{
		procManager: procManager,
		debug:       debug,
	}
}

var _ DigestHelper = &dockerStoreDigest{}

func (d *dockerStoreDigest) PopulateDigest(ctx context.Context, reference *image.Reference) error {
	if reference == nil {
		return nil
	}
	// refString will always have the tag specified at this point.
	// For "scratch", we have to compare it against "scratch:latest" even though
	// scratch:latest isn't valid in a FROM clause.
	if reference.Reference == NoBaseImageSpecifierLatest {
		return nil
	}
	args := []string{
		"docker",
		"run",
		"--rm",

		// Mount home
		"--volume", util.DockerSocketVolumeMapping,
		"--volume", homeVol + ":" + homeWorkDir,
		"--env", homeEnv,

		"docker",
		"inspect",
		"--format",
		"\"{{json .RepoDigests}}\"",
		reference.Reference,
	}
	if d.debug {
		log.Printf("query digest args: %v\n", args)
	}
	var buf bytes.Buffer
	if err := d.procManager.Run(ctx, args, nil, &buf, &buf, ""); err != nil {
		return errors.Wrapf(err, "failed to query digests, msg: %s", buf.String())
	}
	trimCharPredicate := func(c rune) bool {
		return c == '\n' || c == '\r' || c == '"' || c == '\t'
	}
	reference.Digest = getRepoDigest(strings.TrimFunc(buf.String(), trimCharPredicate), reference)
	return nil
}

func getRepoDigest(jsonContent string, reference *image.Reference) string {
	prefix := reference.Repository + "@"
	// If the reference has "library/" prefixed, we have to remove it - otherwise
	// we'll fail to query the digest, since image names aren't prefixed with "library/"
	if strings.HasPrefix(prefix, "library/") && reference.Registry == DockerHubRegistry {
		prefix = prefix[8:]
	} else if len(reference.Registry) > 0 && reference.Registry != DockerHubRegistry {
		prefix = reference.Registry + "/" + prefix
	}
	var digestList []string
	if err := json.Unmarshal([]byte(jsonContent), &digestList); err != nil {
		log.Printf("Error deserializing %s to json, error: %v\n", jsonContent, err)
	}
	for _, digest := range digestList {
		if strings.HasPrefix(digest, prefix) {
			return digest[len(prefix):]
		}
	}
	return ""
}
