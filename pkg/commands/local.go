package commands

import (
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
)

type localSource struct {
	dir     string
	tracker *DirectoryTracker
}

// NewLocalSource creates an object denoting locally mounted source
func NewLocalSource(dir string) domain.BuildSource {
	return &localSource{dir: dir}
}

func (s *localSource) Return(runner domain.Runner) error {
	if s.tracker != nil {
		return s.tracker.Return(runner)
	}
	return nil
}

func (s *localSource) Obtain(runner domain.Runner) (err error) {
	s.tracker, err = ChdirWithTracking(runner, s.dir)
	return
}

func (s *localSource) Export() []domain.EnvVar {
	exports := []domain.EnvVar{}
	if s.dir != "" {
		exports = append(exports, domain.EnvVar{
			Name:  constants.CheckoutDirVar,
			Value: s.dir,
		})
	}
	return exports
}
