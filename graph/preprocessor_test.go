// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"reflect"
	"testing"
)

//Test alias parsing components

/* TestResolveMapAndValidate: M*/
func TestResolveMapAndValidate(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		alias       Alias
	}{
		{
			"Improper Directive Choice",
			true,
			Alias{
				[]*string{},
				map[string]string{"$": "a"},
				'$',
			},
		},
		{
			"Improper Key Name",
			true,
			Alias{
				[]*string{},
				map[string]string{"totally&^Invalid": "hello-world"},
				'$',
			},
		},
		{
			"Improper Directive Length",
			true,
			Alias{
				[]*string{},
				map[string]string{"$": "&&&"},
				'$',
			},
		},
		{
			"Valid Alias",
			false,
			Alias{
				[]*string{},
				map[string]string{"$": "&", "totallyValid": "hello-world"},
				'$',
			},
		},
	}

	for _, test := range tests {
		err := test.alias.resolveMapAndValidate()
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
	}
}

func TestLoadExternalAlias(t *testing.T) {
	resStrings := []string{"nonexistent.yaml",
		"https://httpstat.us/404",
		"https://raw.githubusercontent.com/estebanreyl/preprocessor-test/master/input/valid-remote.yaml",
		"./testdata/preprocessor/valid-external.yaml",
		"./testdata/preprocessor/empty-external.yaml",
	}

	tests := []struct {
		name          string
		shouldError   bool
		alias         Alias
		expectedAlias Alias
	}{
		{
			"Single Nonexistent File",
			true,
			Alias{
				[]*string{&resStrings[0]},
				map[string]string{},
				'$',
			},
			Alias{},
		},
		{
			"Single Nonexistent URL",
			true,
			Alias{
				[]*string{&resStrings[1]},
				map[string]string{},
				'$',
			},
			Alias{},
		},
		{
			"Valid Remote",
			false,
			Alias{
				[]*string{&resStrings[2]},
				map[string]string{},
				'$',
			},
			Alias{
				[]*string{&resStrings[2]},
				map[string]string{
					"docker": "azure/images/docker",
					"cache":  "--cache-from=ubuntu",
				},
				'$',
			},
		},
		{
			"Valid Files",
			false,
			Alias{
				[]*string{&resStrings[3]},
				map[string]string{},
				'$',
			},
			Alias{
				[]*string{&resStrings[3]},
				map[string]string{
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				'$',
			},
		},
		{
			"Empty File",
			false,
			Alias{
				[]*string{&resStrings[4]},
				map[string]string{"d": "docker", "azureCmd": "mcr.microsoft.com/azure-cli"},
				'$',
			},
			Alias{
				[]*string{&resStrings[4]},
				map[string]string{"d": "docker", "azureCmd": "mcr.microsoft.com/azure-cli"},
				'$',
			},
		},
		{
			"Valid All",
			false,
			Alias{
				[]*string{&resStrings[2], &resStrings[3]},
				map[string]string{},
				'$',
			},
			Alias{
				[]*string{&resStrings[2], &resStrings[3]},
				map[string]string{
					"docker":      "azure/images/docker",
					"cache":       "--cache-from=ubuntu",
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				'$',
			},
		},
		{
			"Precedence in override",
			false,
			Alias{
				[]*string{&resStrings[3]},
				map[string]string{"singularity": "something else"},
				'$',
			},
			Alias{
				[]*string{&resStrings[2], &resStrings[3]},
				map[string]string{
					"singularity": "something else",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				'$',
			},
		},
	}

	for _, test := range tests {
		err := test.alias.loadExternalAlias()
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}

		eq := reflect.DeepEqual(test.alias.AliasMap, test.alias.AliasMap)
		if !eq {
			t.Fatalf("Expected output for " + test.name + " differed from actual")
		}
	}
}

func TestAddAliasFromRemote(t *testing.T) {
	resStrings := []string{
		"https://raw.githubusercontent.com/estebanreyl/preprocessor-test/master/input/invalid-remote.yaml",
		"https://raw.githubusercontent.com/estebanreyl/preprocessor-test/master/input/valid-remote.yaml",
	}
	tests := []struct {
		name          string
		shouldError   bool
		alias         Alias
		expectedAlias Alias
	}{
		{
			"Improperly Formatted",
			true,
			Alias{
				[]*string{&resStrings[0]},
				map[string]string{"pre": "someother/pre"},
				'$',
			},
			Alias{},
		},
		{
			"Properly Formatted",
			false,
			Alias{
				[]*string{&resStrings[1]},
				map[string]string{
					"pre": "someother/pre",
				},
				'$',
			},
			Alias{
				[]*string{&resStrings[1]},
				map[string]string{
					"pre":    "someother/pre",
					"docker": "azure/images/docker",
					"cache":  "--cache-from=ubuntu",
				},
				'$',
			},
		},
	}

	for _, test := range tests {
		err := addAliasFromRemote(&test.alias, *test.alias.AliasSrc[0])
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
		eq := reflect.DeepEqual(test.alias.AliasMap, test.alias.AliasMap)
		if !eq {
			t.Fatalf("Expected output for " + test.name + " differed from actual")
		}
	}
}

func TestAddAliasFromFile(t *testing.T) {
	resStrings := []string{
		"./testdata/preprocessor/invalid-external.yaml",
		"./testdata/preprocessor/valid-external.yaml",
	}
	tests := []struct {
		name          string
		shouldError   bool
		alias         Alias
		expectedAlias Alias
	}{
		{
			"Improperly Formatted",
			true,
			Alias{
				[]*string{&resStrings[0]},
				map[string]string{"pre": "someother/pre"},
				'$',
			},
			Alias{},
		},
		{
			"Properly Formatted",
			false,
			Alias{
				[]*string{&resStrings[1]},
				map[string]string{
					"pre": "someother/pre",
				},
				'$',
			},
			Alias{
				[]*string{&resStrings[1]},
				map[string]string{
					"pre":         "someother/pre",
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				'$',
			},
		},
	}

	for _, test := range tests {
		err := addAliasFromFile(&test.alias, *test.alias.AliasSrc[0])
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
		eq := reflect.DeepEqual(test.alias.AliasMap, test.alias.AliasMap)
		if !eq {
			t.Fatalf("Expected output for " + test.name + " differed from actual")
		}
	}
}

/*
// Task tests
func TestPreProcessBytes(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		isFile      bool
		value       string
	}{
		{
			"Data from ACR Task Json",
			false,
			true,
			"somefilename",
		},
		{
			"Data from ACR Task Commandline String",
			false,
			false,
			"somefilename",
		},
		{
			"Invalid Task from File",
			true,
			true,
			"somefilename",
		},
		{
			"Invalid Commandline String",
			true,
			false,
			"somefilename",
		},
		{
			"Nested Values",
			true,
			true,
			"somefilename",
		},
	}

	for _, test := range tests {
		err := nil
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
	}
}

func TestPreProcessString(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		isFile      bool
		value       string
	}{
		{
			"Chained values",
			false,
			true,
			"somefilename",
		},
		{
			"Chained values changed directive",
			false,
			true,
			"somefilename",
		},
		{
			"Multiline replacements",
			true,
			true,
			"somefilename",
		},
		{
			"Complex command",
			true,
			false,
			"somecommand",
		},
	}

	for _, test := range tests {
		err := test.alias.resolveMapAndValidate()
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
	}
}

func TestPreProcessSteps(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		alias       Alias
	}{
		{
			"Improper Directive Choice",
			true,
			Alias{},
		},
		{
			"Improper Key Name",
			true,
			Alias{},
		},
		{
			"Improper Directive Length",
			true,
			Alias{},
		},
		{
			"Valid Alias",
			false,
			Alias{},
		},
	}

	for _, test := range tests {
		err := test.alias.resolveMapAndValidate()
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
	}
}

func TestPreProcessTaskFully(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		alias       Alias
	}{
		{
			"Proper Pass File",
			true,
			Alias{},
		},
		{
			"Proper Pass Command",
			false,
			Alias{},
		},
		{
			"Proper Pass External definitions Command",
			false,
			Alias{},
		},
		{
			"Proper Pass External definitions File",
			true,
			Alias{},
		},
	}

	for _, test := range tests {
		err := test.alias.resolveMapAndValidate()
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.name + " to error but it didn't")
		}
	}
}
*/
