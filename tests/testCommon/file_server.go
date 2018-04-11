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
	startRetryN := 10
	startRetryInterval := time.Second * 10
	go func() {
		retryCount := 0
		started := false
		// NOTE: Sometimes when a test ends and a server shutdown port still appears to be used
		for !started && retryCount < startRetryN {
			err := server.ListenAndServe()
			if err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "Failed to start server: %s\n", err)
				time.Sleep(startRetryInterval)
				retryCount++
			} else {
				started = true
			}
		}
		if !started {
			t.Fail()
		}
	}()

	totalDuration := startRetryInterval * (time.Duration)(startRetryN)
	err := waitForServerReady(totalDuration, time.Second)
	if err != nil {
		t.Fail()
	}

	return server
}

// waitForServerReady waits for server ready until timeout
func waitForServerReady(timeout time.Duration, interval time.Duration) error {

	for remaining := timeout; remaining > 0; remaining -= interval {
		time.Sleep(1 * time.Second)

		_, err := http.Head(StaticFileHost)
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("Server was still not ready after %v seconds", timeout/time.Second)
}

// StaticArchiveHandler streams the hello-multistage project to http response as tar.gz
// We hard code the archive file for now, because we only have 1 test scenario using it
type StaticArchiveHandler struct {
	T           *testing.T
	ArchiveRoot string
}

// ServeHTTP serves content of a directory
func (h *StaticArchiveHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := StreamArchiveFromDir(h.T, h.ArchiveRoot, w)
	if err != nil {
		panic(err.Error())
	}
}

// StreamArchiveFromDir takes a directory and create a tar.gz out of it
func StreamArchiveFromDir(t *testing.T, root string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer ReportOnError(t, gw.Close)
	tw := tar.NewWriter(gw)
	defer ReportOnError(t, tw.Close)

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
		defer ReportOnError(t, file.Close)
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

// FixedResponseHandler serves a fix response
type FixedResponseHandler struct {
	Body         []byte
	ErrorMessage string
	StatusCode   int
}

func (h *FixedResponseHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.ErrorMessage != "" {
		http.Error(w, h.ErrorMessage, h.StatusCode)
	} else {
		w.Write(h.Body)
	}
}
