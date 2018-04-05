package testCommon

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

const staticFilePort = ":32764"

// StaticFileHost is where the file server is
const StaticFileHost = "http://localhost" + staticFilePort

// StartStaticFileServer starts a file server, if error returned
// is not null the caller should close it
func StartStaticFileServer(t *testing.T, handler http.Handler) *http.Server {
	server := &http.Server{
		Addr:    staticFilePort,
		Handler: handler,
	}
	go func() {
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			t.Errorf("Failed to start server: %s", err)
			t.Fail()
		}
	}()

	err := WaitForServerReady(10)
	if err != nil {
		t.Fail()
	}

	return server
}

// WaitForServerReady waits for server ready until timeout
func WaitForServerReady(timeoutInSeconds uint) error {

	for i := uint(0); i < timeoutInSeconds; i++ {
		time.Sleep(1 * time.Second)

		_, err := http.Head(StaticFileHost)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("Server was still not ready after %v seconds", timeoutInSeconds)
}

// StaticArchiveHandler streams the hello-multistage project to http response as tar.gz
// We hard code the archive file for now, because we only have 1 test scenario using it
type StaticArchiveHandler struct {
	T           *testing.T
	ArchiveRoot string
}

// ServeHTTP serves content of a directory
func (h *StaticArchiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.streamArchive(h.ArchiveRoot, w)
	if err != nil {
		panic(err.Error())
	}
}

func (h *StaticArchiveHandler) streamArchive(root string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer ReportOnError(h.T, gw.Close)
	tw := tar.NewWriter(gw)
	defer ReportOnError(h.T, tw.Close)

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
		defer ReportOnError(h.T, file.Close)
		_, err = io.Copy(tw, file)
		return err
	})
}

// StaticContentHandler serves a fixed string
type StaticContentHandler struct {
	T       *testing.T
	Content []byte
}

// ServeHTTP serves the static contents
func (h *StaticContentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write(h.Content); err != nil {
		panic(err)
	}
}
