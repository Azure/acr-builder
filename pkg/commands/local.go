package commands

import (
	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
)

type localSource struct {
	dir string
}

// NewLocalSource creates an object denoting locally mounted source
func NewLocalSource(dir string) (domain.BuildSource, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("Error trying to locate current working directory as local source %s", err)
		}
	}
	return &localSource{dir: dir}, nil
}

func (s *localSource) Obtain(runner domain.Runner) error {
	fs := runner.GetFileSystem()
	return fs.Chdir(s.dir)
}

func (s *localSource) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.CheckoutDirVar,
			Value: s.dir,
		},
	}
}
