package testCommon

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

const staticFilePort = ":32764"

// StaticFileHost is where the file server is
const StaticFileHost = "http://localhost" + staticFilePort

// StartStaticFileServer starts a file server, if error returned
// is not null the caller should close it
func StartStaticFileServer(t *testing.T) *http.Server {
	server := &http.Server{
		Addr:    staticFilePort,
		Handler: &staticFileArchiveHandler{},
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start server: %s", err)
			t.Fail()
		}
	}()
	return server
}

// staticFileArchiveHandler streams the docker-compose project to http response as tar.gz
// We hard code the archive file for now, because we only have 1 test scenario using it
type staticFileArchiveHandler struct {
	t *testing.T
}

func (h *staticFileArchiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	projectRoot := filepath.Join(Config.ProjectRoot, "tests", "resources", "docker-compose")
	err := h.streamArchive(projectRoot, w)
	if err != nil {
		panic(err.Error())
	}
}

func (h *staticFileArchiveHandler) streamArchive(root string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer ReportOnError(h.t, gw.Close)
	tw := tar.NewWriter(gw)
	defer ReportOnError(h.t, tw.Close)

	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, info.Name())
		if err != nil {
			return err
		}

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer ReportOnError(h.t, file.Close)
		_, err = io.Copy(tw, file)
		return err
	})
}
