// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/pkg/errors"
)

// Test alias parsing components
// Under consideration, multi alphabet alias support?
// yamlReserved... Note there are escapes for all
//  := c-indicator	::=	  “-” | “?” | “:” | “,” | “[” | “]” | “{” | “}”
// | “#” | “&” | “*” | “!” | “|” | “>” | “'” | “"”
// | “%” | “@” | “`”
// ( b-carriage-return b-line-feed )  DOS, Windows
// | b-carriage-return MacOS upto 9.x
// | b-line-feed   UNIX, MacOS X
// s-space	::=	#x20  SP
// s-tab	::=	#x9  TAB

// TestResolveMapAndValidate: Will make sure ResolveMapAndValidate is performing as expected
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
				[]string{},
				map[string]string{},
				"a",
				'$',
			},
		},
		{
			"Improper Key Name",
			true,
			Alias{
				[]string{},
				map[string]string{"totally&^Invalid": "hello-world"},
				"$",
				'$',
			},
		},
		{
			"Improper Directive Length",
			true,
			Alias{
				[]string{},
				map[string]string{},
				"&&&",
				'$',
			},
		},
		{
			"Valid Alias",
			false,
			Alias{
				[]string{},
				map[string]string{"totallyValid": "hello-world"},
				"&",
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

// TestLoadExternalAlias: Makes sure Loading an external Alias works as expected
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
				[]string{resStrings[0]},
				map[string]string{},
				"$",
				'$',
			},
			Alias{},
		},
		{
			"Single Nonexistent URL",
			true,
			Alias{
				[]string{resStrings[1]},
				map[string]string{},
				"$",
				'$',
			},
			Alias{},
		},
		{
			"Valid Remote",
			false,
			Alias{
				[]string{resStrings[2]},
				map[string]string{},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[2]},
				map[string]string{
					"docker": "azure/images/docker",
					"cache":  "--cache-from=ubuntu",
				},
				"$",
				'$',
			},
		},
		{
			"Valid Files",
			false,
			Alias{
				[]string{resStrings[3]},
				map[string]string{},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[3]},
				map[string]string{
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				"$",
				'$',
			},
		},
		{
			"Empty File",
			false,
			Alias{
				[]string{resStrings[4]},
				map[string]string{"d": "docker", "azureCmd": "mcr.microsoft.com/azure-cli"},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[4]},
				map[string]string{"d": "docker", "azureCmd": "mcr.microsoft.com/azure-cli"},
				"$",
				'$',
			},
		},
		{
			"Valid All",
			false,
			Alias{
				[]string{resStrings[2], resStrings[3]},
				map[string]string{},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[2], resStrings[3]},
				map[string]string{
					"docker":      "azure/images/docker",
					"cache":       "--cache-from=ubuntu",
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				"$",
				'$',
			},
		},
		{
			"Precedence in override",
			false,
			Alias{
				[]string{resStrings[3]},
				map[string]string{"singularity": "something else"},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[2], resStrings[3]},
				map[string]string{
					"singularity": "something else",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				"$",
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
				[]string{resStrings[0]},
				map[string]string{"pre": "someother/pre"},
				"$",
				'$',
			},
			Alias{},
		},
		{
			"Properly Formatted",
			false,
			Alias{
				[]string{resStrings[1]},
				map[string]string{
					"pre": "someother/pre",
				},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[1]},
				map[string]string{
					"pre":    "someother/pre",
					"docker": "azure/images/docker",
					"cache":  "--cache-from=ubuntu",
				},
				"$",
				'$',
			},
		},
	}

	for _, test := range tests {
		err := addAliasFromRemote(&test.alias, test.alias.AliasSrc[0])
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
				[]string{resStrings[0]},
				map[string]string{"pre": "someother/pre"},
				"$",
				'$',
			},
			Alias{},
		},
		{
			"Properly Formatted",
			false,
			Alias{
				[]string{resStrings[1]},
				map[string]string{
					"pre": "someother/pre",
				},
				"$",
				'$',
			},
			Alias{
				[]string{resStrings[1]},
				map[string]string{
					"pre":         "someother/pre",
					"singularity": "mcr.microsoft.com/acr-task-commands/singularity-builder:3.3",
					"pack":        "mcr.microsoft.com/azure-task-commands/buildpack:latest pack",
					"git":         "azure/images/git",
				},
				"$",
				'$',
			},
		},
	}

	for _, test := range tests {
		err := addAliasFromFile(&test.alias, test.alias.AliasSrc[0])
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

// Task with processing tests

func TestPreProcessBytes(t *testing.T) {
	taskDefinitionSrc := "./testdata/preprocessor/preprocessing-stress.yaml"
	yamlMap, err := extractTaskYamls(taskDefinitionSrc)
	if err != nil {
		t.Fatalf("Could not read source for tests at:" + taskDefinitionSrc + "Error: " + err.Error())
	}
	tests := []struct {
		nameAndTaskIdentifier string
		expected              string
		shouldError           bool
		description           string
	}{
		{
			"Chaining Directive Unicode",
			"Expected",
			false,
			"Tests multiple single letter aliases using an overwritten directive",
		},
		{
			"Multiline Alias",
			"Expected",
			false,
			"Tests the edge case where a user specifies a multiline alias",
		},
		{
			"Escape",
			"Expected Escape",
			false,
			"Verifies escape sequences work correctly",
		},
		{
			"Expected",
			"Expected",
			false,
			"Makes sure files with no Alias remain unaffected",
		},
		{
			"Alias No Use",
			"Expected",
			false,
			"Makes sure if aliases are defined and unused no unexpected changes will happen",
		},
	}

	for _, test := range tests {
		input := yamlMap[test.nameAndTaskIdentifier]
		data, _, _, err := PreprocessBytes(input, nil)
		if err != nil && test.shouldError {
			continue
		}
		if err == nil && test.shouldError {
			t.Fatalf("Expected test " + test.nameAndTaskIdentifier + " to error but it didn't")
		}
		if err != nil {
			t.Fatalf("Test " + test.nameAndTaskIdentifier + "failed with error: " + err.Error())
		}
		var actual interface{}
		yaml.Unmarshal(data, &actual)
		actualBytes, err := yaml.Marshal(actual)
		if err != nil {
			t.Fatalf("Test " + test.nameAndTaskIdentifier + "failed with error: " + err.Error())
		}
		var expected interface{}
		yaml.Unmarshal(yamlMap[test.expected], &expected)
		expectedBytes, err := yaml.Marshal(expected)
		if err != nil {
			t.Fatalf("Test " + test.nameAndTaskIdentifier + "failed with error: " + err.Error())
		}

		if !bytes.Equal(actualBytes, expectedBytes) {
			fmt.Print("Actual: \n")
			fmt.Print(string(data))
			fmt.Print("Expected: \n")
			fmt.Print(string(yamlMap[test.expected]))
			t.Fatalf("Expected output for " + test.nameAndTaskIdentifier + " differed from actual")
		}

	}
}

func extractTaskYamls(file string) (map[string][]byte, error) {
	processed := make(map[string][]byte)
	data, fileReadingError := ioutil.ReadFile(file)

	if fileReadingError != nil {
		return processed, fileReadingError
	}

	reader := bytes.NewReader(data)
	scanner := bufio.NewScanner(reader)
	scanner.Split(bufio.ScanLines)

	var curBuffer bytes.Buffer

	genericTopLevelRe := regexp.MustCompile(`\A[^\s:]+[^:]*:.*\z`)
	cleanName := regexp.MustCompile(`[^\s:]+[^:]*`)

	current := "Comments"
	for scanner.Scan() {
		text := scanner.Text()
		if matched := genericTopLevelRe.MatchString(text); matched {
			// Top level item has already been seen, this is not allowed
			if _, ok := processed[current]; ok {
				return processed, errors.New("duplicate top level testing yaml was declared")
			}

			processed[current] = make([]byte, len(curBuffer.Bytes()))
			copy(processed[current], curBuffer.Bytes())
			curBuffer.Reset()
			current = strings.Trim(cleanName.FindString(text), " ")
		} else {
			if len(text) >= 2 {
				text = text[2:] // Remove spacing offset
			}
			curBuffer.WriteString(text + "\n")
		}
	}
	processed[current] = make([]byte, len(curBuffer.Bytes()))
	copy(processed[current], curBuffer.Bytes())
	return processed, nil
}

func TestPreProcessSteps(t *testing.T) {
	resSteps := []Step{
		{Cmd: `purge --registry {{.Run.Registry}} --filter 'samples/devimage1:.*' --filter 'samples/devimage2:.*' --ago 0d --untagged --dry-run"`},
		{Cmd: `        purge --registry {{.Run.Registry}} --filter 'samples/devimage1:.*' --filter 'samples/devimage2:.*' --ago 0d --untagged --dry-run"            `},
		{Cmd: `fakealias --wait 300`},
		{Cmd: `acrmixedin --wait 300`},
		{Cmd: `acr purge --registry {{.Run.Registry}} --filter 'samples/devimage1:.*' --filter 'samples/devimage2:.*' --ago 0d --untagged --dry-run"`},
	}
	tests := []struct {
		nameAndTaskIdentifier string
		shouldError           bool
		alias                 Alias
		current               Task
		expected              Step
	}{
		{
			"Simple replacement",
			false,
			Alias{
				AliasMap: map[string]string{"purge": "acr purge"},
			},
			Task{
				Steps: []*Step{&resSteps[0]},
			},
			resSteps[4],
		},
		{
			"Spaces untrimmed",
			false,
			Alias{
				AliasMap: map[string]string{"purge": "acr purge"},
			},
			Task{
				Steps: []*Step{&resSteps[1]},
			},
			resSteps[4],
		},
		{
			"Non-existent alias",
			false,
			Alias{
				AliasMap: map[string]string{"acr": "acr expanded"},
			},
			Task{
				Steps: []*Step{&resSteps[2]},
			},
			resSteps[2],
		},
		{
			"Alias is substring",
			false,
			Alias{
				AliasMap: map[string]string{"acr": "acr expanded"},
			},
			Task{
				Steps: []*Step{&resSteps[3]},
			},
			resSteps[3],
		},
	}

	for _, test := range tests {
		ExpandCommandAliases(&test.alias, &test.current)

		if test.current.Steps[0].Cmd != test.expected.Cmd {
			t.Fatalf("Test " + test.nameAndTaskIdentifier + " expected: " + test.expected.Cmd + " but resolved to " + test.current.Steps[0].Cmd)
		}
	}
}

func TestFindVersion(t *testing.T) {
	tests := []struct {
		task            string
		expectedVersion string
	}{
		{
			"",
			"",
		},
		{
			` # task.yml file
# should be skipped

version  : v1.1.0    
`,
			"v1.1.0",
		},
		{
			`
build: something
version: v1.1.0`,
			"",
		},
		{
			`
versionOfTask:v1.1.0`,
			"",
		},
		{
			`       
version`,
			"",
		},
		{
			`       
version:v1.1.0
			`,
			"v1.1.0",
		},
		{
			`       
version:foo:bar:beta`,
			"foo:bar:beta",
		},
	}

	for _, test := range tests {
		actualVersion := FindVersion([]byte(test.task))
		if actualVersion != test.expectedVersion {
			t.Errorf("Expected %s but got %s", test.expectedVersion, actualVersion)
		}
	}
}
