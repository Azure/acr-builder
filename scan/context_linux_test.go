// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"archive/tar"
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestGetContextFromReaderUntarsDeviceNode(t *testing.T) {
	archive := createContextTar(t, []contextTarEntry{
		{header: tar.Header{Name: "dev/ptmx", Typeflag: tar.TypeChar, Mode: 0640, Devmajor: 5, Devminor: 2}},
	})
	destination := t.TempDir()
	scanner := &Scanner{destinationFolder: destination}
	if err := scanner.getContextFromReader(bytes.NewReader(archive)); err != nil {
		// Creating device nodes requires CAP_MKNOD, which ordinary test runners lack.
		if errors.Is(err, syscall.EPERM) || errors.Is(err, syscall.EACCES) {
			t.Skipf("device creation is not permitted: %v", err)
		}
		t.Fatal(err)
	}
	info, err := os.Lstat(filepath.Join(destination, "dev", "ptmx"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeCharDevice == 0 || info.Mode().Perm() != 0640 {
		t.Fatalf("extracted device mode = %v, want character device with mode 0640", info.Mode())
	}
}
