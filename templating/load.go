// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// LoadConfig creates a Config from the specified path.
func LoadConfig(path string) (*Config, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}

	return &Config{RawValue: string(data)}, nil
}

// LoadTemplate loads a Template from the specified path.
func LoadTemplate(path string) (*Template, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}

	return &Template{Name: path, Data: data}, nil
}

func readFile(path string) ([]byte, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(abs)
	if err != nil {
		return nil, err
	}

	data, err := ioutil.ReadFile(abs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read file: %s, absolute path: %s", path, abs)
	}

	return data, nil
}
