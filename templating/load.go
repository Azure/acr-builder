// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const (
	decodedTemplateName = "decoded"
)

// LoadConfig creates a Config from the specified path.
func LoadConfig(path string) (*Config, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load values file at path %s", path)
	}

	return &Config{RawValue: string(data)}, nil
}

// DecodeConfig loads a Config from a Base64 encoded string.
func DecodeConfig(encoded string) (*Config, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Base64 config")
	}
	return &Config{RawValue: string(decoded)}, nil
}

// LoadTemplate loads a Template from the specified path.
func LoadTemplate(path string) (*Template, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load template at path %s", path)
	}

	return NewTemplate(path, data), nil
}

// DecodeTemplate loads a Template from a Base64 encoded string.
func DecodeTemplate(encoded string) (*Template, error) {
	decoded, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode Base64 template")
	}
	return NewTemplate(decodedTemplateName, decoded), nil
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
