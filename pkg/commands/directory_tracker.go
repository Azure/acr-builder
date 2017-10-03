package commands

import (
	"github.com/Azure/acr-builder/pkg/domain"
)

type DirectoryTracker struct {
	path     string
	tracking *DirectoryTracker
}

func ChdirWithTracking(runner domain.Runner, chdir string) (*DirectoryTracker, error) {
	if chdir == "" {
		return nil, nil
	}
	fs := runner.GetFileSystem()
	path, err := fs.Getwd()
	if err != nil {
		return nil, err
	}
	err = fs.Chdir(chdir)
	if err != nil {
		return nil, err
	}
	return &DirectoryTracker{path: path}, nil
}

func (t *DirectoryTracker) Return(runner domain.Runner) error {
	fs := runner.GetFileSystem()
	return fs.Chdir(t.path)
}
