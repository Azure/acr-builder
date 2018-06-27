// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// LoadJob loads a job from the specified path.
func LoadJob(path string) (*Job, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}

	if fi.IsDir() {
		if validJob, err := IsValidJobDir(abs); !validJob {
			return nil, err
		}

		return LoadJobFromDir(path)
	}

	return nil, fmt.Errorf("Unable to load job from path: %s", path)
}

// IsValidJobDir determines whether or not the specified file info describes a valid Job.
func IsValidJobDir(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return false, err
	}

	if !fi.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}

	return true, nil
}

// LoadJobFromDir loads a job from the specified directory.
func LoadJobFromDir(path string) (*Job, error) {
	files := []*InMemoryFile{}
	j := &Job{}

	// TODO: support symlinks?
	path += string(filepath.Separator)
	err := filepath.Walk(path, func(name string, fi os.FileInfo, err error) error {

		n := strings.TrimPrefix(name, path)
		// Don't process the top dir.
		if n == "" {
			return nil
		}

		// Normalize slashes
		n = filepath.ToSlash(n)

		// Skip directories
		if !fi.IsDir() {
			data, err := ioutil.ReadFile(name)
			if err != nil {
				return fmt.Errorf("Failed to read %s. Err: %v", name, err)
			}
			files = append(files, NewInMemoryFile(n, data))
		}
		return nil
	})

	if err != nil {
		return j, err
	}

	for _, f := range files {
		if f.Name == "values.toml" {
			j.Config = &Config{RawValue: string(f.Data)}
		} else if strings.HasPrefix(f.Name, "templates/") {
			j.Templates = append(j.Templates, &Template{Name: f.Name, Data: f.Data})
		}
	}

	return j, nil
}
