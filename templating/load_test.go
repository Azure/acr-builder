// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"testing"
)

const (
	thamesPath = "testdata/thames"
)

// TestLoadJobFromDir tests loading a Job from a directory with templates
func TestLoadJobFromDir(t *testing.T) {
	j, err := LoadJob(thamesPath)
	if err != nil {
		t.Fatalf("Failed to load job at path %s. Err: %v", thamesPath, err)
	}
	if j == nil {
		t.Fatalf("Job at path %s was nil.", thamesPath)
	}

	if j.Config == nil {
		t.Fatalf("Job config is nil")
	}

	expectedNumTmpls := 2
	expectedTemplateNames := map[string]bool{"templates/pre-thames.toml": true, "templates/thames.toml": true}

	// Verify templates
	if len(j.Templates) != expectedNumTmpls {
		t.Errorf("Expected %d templates but got %d", expectedNumTmpls, len(j.Templates))
	}

	for _, v := range j.Templates {
		if ok := expectedTemplateNames[v.Name]; !ok {
			t.Errorf("%s was not an expected template", v.Name)
		}
	}

}

// TestIsValidJobDir_Valid ensures that a valid job directory is marked as such.
func TestIsValidJobDir_Valid(t *testing.T) {
	j, err := IsValidJobDir(thamesPath)
	if err != nil {
		t.Fatalf("Unexpected error during job dir validation: %v", err)
	}

	if !j {
		t.Fatalf("Expected %s to be a valid job dir", thamesPath)
	}
}
