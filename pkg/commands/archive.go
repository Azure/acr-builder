package commands

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
)

const maxHeaderSize = 4

// NOTE: only support .tar.gz for now
var supportedArchiveHeaders = map[byte][]byte{
	// Gzip
	0x1F: {0x1F, 0x8B, 0x08},
}

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

	buf := bufio.NewReader(response.Body)
	peek, err := buf.Peek(maxHeaderSize)
	if err != nil && err != io.EOF {
		return fmt.Errorf("Failed to peek context header: %s", err)
	}
	supported, err := isSupportedArchive(peek)
	if err != nil {
		return fmt.Errorf("Failed to read context header: %s", err)
	}
	if !supported {
		return fmt.Errorf("Unexpected file format for %s", s.url)
	}

	baseDir := s.targetDir
	if baseDir == "" {
		baseDir = "."
	}
	if err := unTAR(baseDir, buf); err != nil {
		return fmt.Errorf("Failed to extract archive from %s, error: %s", s.url, err)
	}

	tracker, err := ChdirWithTracking(runner, s.targetDir)
	if err != nil {
		return err
	}
	s.tracker = tracker
	return nil
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

func unTAR(baseDir string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	defer func() {
		err := gzr.Close()
		if err != nil {
			logrus.Warnf("Error closing gz archive, error: %s", err)
		}
	}()
	if err != nil {
		return err
	}
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil

		case err != nil:
			return err

		case header == nil:
			continue
		}

		target := filepath.Join(baseDir, header.Name)

		// check the file type
		switch header.Typeflag {

		// if its a dir and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return err
				}
			}

		// if it's a file create it
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			defer func() {
				err := f.Close()
				if err != nil {
					logrus.Warnf("Error closing file %s, error: %s", target, err)
				}
			}()
			// copy over contents
			if _, err := io.Copy(f, tr); err != nil {
				return err
			}

		default:
			logrus.Debugf("Ignoring unexpected file %s, type: %d", header.Name, header.Typeflag)
		}
	}
}

func isSupportedArchive(source []byte) (bool, error) {
	if len(source) < 1 {
		return false, fmt.Errorf("Empty header")
	}
	header, found := supportedArchiveHeaders[source[0]]
	if len(source) < len(header) {
		return false, fmt.Errorf("Format of the file content cannot be determined. File header is corrupted")
	}
	return found && bytes.Equal(source[:len(header)], header), nil
}
