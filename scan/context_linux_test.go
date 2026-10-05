// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestGetContextFromReaderUntarsDeviceNode(t *testing.T) {
	archive := createContextTar(t, []contextTarEntry{
		{header: tar.Header{Name: "dev/ptmx", Typeflag: tar.TypeChar, Mode: 0640, Devmajor: 5, Devminor: 2}},
	})
	destination := t.TempDir()
	scanner := &Scanner{destinationFolder: destination}
	if err := scanner.getContextFromReader(bytes.NewReader(archive)); err != nil {
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
