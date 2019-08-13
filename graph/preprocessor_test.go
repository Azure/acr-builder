// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"testing"

	yaml "gopkg.in/yaml.v2"
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
		if err != nil {
			t.Fatalf("Test " + test.name + "failed with error: " + err.Error())
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

		if err != nil {
			t.Fatalf("Test " + test.name + "failed with error: " + err.Error())
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

		if err != nil {
			t.Fatalf("Test " + test.name + "failed with error: " + err.Error())
		}

		eq := reflect.DeepEqual(test.alias.AliasMap, test.alias.AliasMap)
		if !eq {
			t.Fatalf("Expected output for " + test.name + " differed from actual")
		}
	}
}

// Task tests

func TestPreProcessBytes(t *testing.T) {
	taskDefinitionSrc := "./testdata/preprocessor/preprocessing-stress.yaml"
	yamlMap, err := extractTaskYamlsAsBytes(taskDefinitionSrc)
	if err != nil {
		t.Fatalf("Could not read source for tests at:" + taskDefinitionSrc + "Error: " + err.Error())
	}
	tests := []struct {
		nameAndTaskIdentifier string
		shouldError           bool
		description           string
	}{
		{
			"Chaining",
			false,
			"Tests 700+ chained aliases",
		},
		{
			"Chaining Directive Changed",
			false,
			"Identical to Chaining but using a redefined directive",
		},
		{
			"Chaining Directive Unicode",
			false,
			"Identical to Chaining but using a redefined Unicode directive",
		},
		{
			"Multiline Alias",
			false,
			"somefilename",
		},
		// {
		// 	"Nested Values",
		// 	true,
		// 	"somefilename",
		// },
		// {
		// 	"Data from ACR Task Json",
		// 	false,
		// 	"somefilename",
		// },
		// {
		// 	"Invalid Task from File",
		// 	true,
		// 	"somefilename",
		// },
		// {
		// 	"Invalid Commandline String",
		// 	true,
		// 	"somefilename",
		// },
		// {
		// 	"Nested Values",
		// 	true,
		// 	"somefilename",
		// },
		// {
		// 	"No Alias",
		// 	true,
		// 	"somefilename",
		// },
		// {
		// 	"Alias No Use",
		// 	true,
		// 	"somefilename",
		// },
	}

	for _, test := range tests {
		input := yamlMap[test.nameAndTaskIdentifier]
		data, _, _, err := preprocessBytes(input)
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.nameAndTaskIdentifier + " to error but it didn't")
		}
		if err != nil {
			t.Fatalf("Test " + test.nameAndTaskIdentifier + "failed with error: " + err.Error())
		}

		if !bytes.Equal(data, yamlMap["Expected"]) {
			fmt.Print("Actual: \n")
			fmt.Print(string(data))
			fmt.Print("Expected: \n")
			fmt.Print(string(yamlMap["Expected"]))
			t.Fatalf("Expected output for " + test.nameAndTaskIdentifier + " differed from actual")
		}

	}
}

func extractTaskYamlsAsBytes(file string) (map[string][]byte, error) {
	processed := make(map[string][]byte)
	var config map[string]interface{}

	data, fileReadingError := ioutil.ReadFile(file)
	if fileReadingError != nil {
		return processed, fileReadingError
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return processed, err
	}

	for k, v := range config {
		var err error
		processed[k], err = yaml.Marshal(v)

		if err != nil {
			return processed, err
		}

	}
	return processed, nil
}

/*
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
			"Proper Pass Full",
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
