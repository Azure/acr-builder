package commands

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/docker/docker/pkg/archive"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/sirupsen/logrus"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
)

const (
	tempWorkingDir = "temp"
	tempFileName   = "temp.tar.gz"
)

// ArchiveSource defines source in the form of an archive file
// Currently we only support tar.gz
type ArchiveSource struct {
	url       string
	targetDir string
	tracker   *DirectoryTracker
}

// NewArchiveSource creates a new archive source
func NewArchiveSource(url string, targetDir string) build.Source {
	return &ArchiveSource{url: url, targetDir: targetDir}
}

// Obtain downloads and extract the source
func (s *ArchiveSource) Obtain(runner build.Runner) error {
	response, err := http.Get(s.url)
	if err != nil {
		return fmt.Errorf("Failed to get archive file from %s, error: %s", s.url, err)
	}
	defer func() {
		err := response.Body.Close()
		if err != nil {
			logrus.Warnf("Error closing http response from archive download, url: %s, error: %s", s.url, err)
		}
	}()

	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	baseDir := s.targetDir
	if baseDir == "" {
		baseDir = tempWorkingDir
	}

	if err = fileutils.CreateIfNotExists(baseDir, true); err != nil {
		return err
	}

	logrus.Infof("Base directory: %s", baseDir)

	fn := tempFileName
	logrus.Infof("Writing file: %s", fn)
	if err = ioutil.WriteFile(fn, bytes, 0755); err != nil {
		return err
	}

	logrus.Infof("Opening tar: %s", fn)
	tarArchive, err := os.Open(fn)
	if err != nil {
		return err
	}
	defer tarArchive.Close()

	// Remove the temporary tar if possible.
	_ = os.Remove(fn)

	// TODO: clean up the untarred directory. It needs to be cleaned up
	// after generating dependencies.
	logrus.Infof("Untarring %s to %s", fn, baseDir)
	if err = archive.Untar(tarArchive, baseDir, nil); err != nil {
		return err
	}

	tracker, err := ChdirWithTracking(runner, baseDir)
	if err != nil {
		return err
	}

	s.tracker = tracker
	return err
}

// Return chdir back, currently not delete the extacted source
func (s *ArchiveSource) Return(runner build.Runner) error {
	if s.tracker != nil {
		return s.tracker.Return(runner)
	}
	return nil
}

// Export exports the variables
func (s *ArchiveSource) Export() []build.EnvVar {
	exports := []build.EnvVar{}
	if s.targetDir != "" {
		exports = append(exports, build.EnvVar{
			Name:  constants.ExportsWorkingDir,
			Value: s.targetDir,
		})
	}
	return exports
}
