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

func TestGetContextFromReaderCreatesImpliedParentForDirectory(t *testing.T) {
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)

	// The archive has no entry for build/ before the build/server/ directory.
	if err := tw.WriteHeader(&tar.Header{Name: "build/server/", Typeflag: tar.TypeDir, Mode: 0755, Uid: os.Getuid(), Gid: os.Getgid()}); err != nil {
		t.Fatal(err)
	}
	const contents = "FROM scratch\n"
	if err := tw.WriteHeader(&tar.Header{Name: "build/server/Dockerfile", Typeflag: tar.TypeReg, Mode: 0644, Size: int64(len(contents)), Uid: os.Getuid(), Gid: os.Getgid()}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(contents)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	destination := t.TempDir()
	scanner := &Scanner{destinationFolder: destination}
	if err := scanner.getContextFromReader(&archive); err != nil {
		t.Fatalf("extract context with an implied parent directory: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(destination, "build", "server", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Fatalf("Dockerfile contents = %q, want %q", got, contents)
	}
}
