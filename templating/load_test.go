// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"io/ioutil"
	"testing"
)

const (
	thamesValuesPath = "testdata/thames/values.toml"
	thamesPath       = "testdata/thames/thames.toml"
)

// TestLoadConfig tests loading a Config.
func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig(thamesValuesPath)
	if err != nil {
		t.Fatalf("failed to load config from path %s. Err: %v", thamesValuesPath, err)
	}

	if c == nil {
		t.Fatalf("resulting config at path %s was nil", thamesValuesPath)
	}

	data, err := ioutil.ReadFile(thamesValuesPath)
	if err != nil {
		t.Fatalf("failed to read file %s. Err: %v", thamesValuesPath, err)
	}

	expected := string(data)
	if expected != c.GetRawValue() {
		t.Fatalf("expected %s but got %s", expected, c.GetRawValue())
	}
}

// TestLoadTemplate tests loading a Template.
func TestLoadTemplate(t *testing.T) {
	template, err := LoadTemplate(thamesPath)
	if err != nil {
		t.Fatalf("failed to load config from path %s. Err: %v", thamesPath, err)
	}

	if template == nil {
		t.Fatalf("resulting template at path %s was nil", thamesPath)
	}

	data, err := ioutil.ReadFile(thamesPath)
	if err != nil {
		t.Fatalf("failed to read file %s. Err: %v", thamesPath, err)
	}

	expectedData := string(data)
	actualData := string(template.GetData())
	if expectedData != actualData {
		t.Fatalf("expected %s as the data but got %s", expectedData, actualData)
	}

	expectedName := thamesPath
	if expectedName != template.GetName() {
		t.Fatalf("expected %s as the template's name but got %s", expectedName, template.GetName())
	}
}
