// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"archive/tar"
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetContextFromReaderCreatesImpliedParentForDirectory(t *testing.T) {
	const contents = "FROM scratch\n"
	archive := createContextTar(t, []contextTarEntry{
		{header: tar.Header{Name: "etc/dnf/", Typeflag: tar.TypeDir, Mode: 0755}},
		{header: tar.Header{Name: "build/server/deep/", Typeflag: tar.TypeDir, Mode: 0755}},
		{header: tar.Header{Name: "build/server/deep/Dockerfile", Typeflag: tar.TypeReg, Mode: 0644}, content: contents},
		{header: tar.Header{Name: "src/FieldLevel.SchemaChangeController/marker", Typeflag: tar.TypeReg, Mode: 0644}, content: "present"},
	})

	destination := t.TempDir()
	scanner := &Scanner{destinationFolder: destination}
	if err := scanner.getContextFromReader(bytes.NewReader(archive)); err != nil {
		t.Fatalf("extract context with an implied parent directory: %v", err)
	}

	if info, err := os.Stat(filepath.Join(destination, "etc", "dnf")); err != nil || !info.IsDir() {
		t.Fatalf("etc/dnf was not extracted as a directory: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(destination, "build", "server", "deep", "Dockerfile"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Fatalf("Dockerfile contents = %q, want %q", got, contents)
	}
	got, err = os.ReadFile(filepath.Join(destination, "src", "FieldLevel.SchemaChangeController", "marker"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "present" {
		t.Fatalf("marker contents = %q, want present", got)
	}
}

func TestGetContextFromReaderUntarsAbsoluteLinks(t *testing.T) {
	t.Run("symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows does not support absolute Unix symlinks")
		}
		archive := createContextTar(t, []contextTarEntry{
			{header: tar.Header{Name: "var/run", Typeflag: tar.TypeSymlink, Linkname: "/run", Mode: 0777}},
			{header: tar.Header{Name: "var/run/existing/non-existing/file", Typeflag: tar.TypeReg, Mode: 0644}, content: "content"},
		})
		destination := t.TempDir()
		scanner := &Scanner{destinationFolder: destination}
		if err := scanner.getContextFromReader(bytes.NewReader(archive)); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(destination, "run", "existing", "non-existing", "file"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "content" {
			t.Fatalf("extracted content = %q, want %q", got, "content")
		}
		target, err := os.Readlink(filepath.Join(destination, "var", "run"))
		if err != nil {
			t.Fatal(err)
		}
		if target != "/run" {
			t.Fatalf("symlink target = %q, want /run", target)
		}
	})

	t.Run("hardlink", func(t *testing.T) {
		archive := createContextTar(t, []contextTarEntry{
			{header: tar.Header{Name: "usr/bin/perlbug", Typeflag: tar.TypeReg, Mode: 0755}, content: "hello"},
			{header: tar.Header{Name: "usr/bin/perlthanks", Typeflag: tar.TypeLink, Linkname: "/usr/bin/perlbug", Mode: 0755}},
		})
		destination := t.TempDir()
		scanner := &Scanner{destinationFolder: destination}
		if err := scanner.getContextFromReader(bytes.NewReader(archive)); err != nil {
			t.Fatal(err)
		}
		target, err := os.Stat(filepath.Join(destination, "usr", "bin", "perlbug"))
		if err != nil {
			t.Fatal(err)
		}
		link, err := os.Stat(filepath.Join(destination, "usr", "bin", "perlthanks"))
		if err != nil {
			t.Fatal(err)
		}
		if !os.SameFile(target, link) {
			t.Fatal("absolute-target hardlink does not refer to the extracted file")
		}
	})
}

type contextTarEntry struct {
	header  tar.Header
	content string
}

func createContextTar(t *testing.T, entries []contextTarEntry) []byte {
	t.Helper()
	var archive bytes.Buffer
	tw := tar.NewWriter(&archive)
	for _, entry := range entries {
		entry.header.Size = int64(len(entry.content))
		entry.header.Uid = os.Getuid()
		entry.header.Gid = os.Getgid()
		if err := tw.WriteHeader(&entry.header); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(entry.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	return archive.Bytes()
}
