package commands

import (
	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
)

type localSource struct {
	dir     string
	tracker *DirectoryTracker
}

// NewLocalSource creates an object denoting locally mounted source
func NewLocalSource(dir string) build.Source {
	return &localSource{dir: dir}
}

func (s *localSource) Return(runner build.Runner) error {
	if s.tracker != nil {
		return s.tracker.Return(runner)
	}
	return nil
}

func (s *localSource) Obtain(runner build.Runner) (err error) {
	s.tracker, err = ChdirWithTracking(runner, s.dir)
	return
}

func (s *localSource) Export() []build.EnvVar {
	exports := []build.EnvVar{}
	if s.dir != "" {
		exports = append(exports, build.EnvVar{
			Name:  constants.ExportsWorkingDir,
			Value: s.dir,
		})
	}
	return exports
}
