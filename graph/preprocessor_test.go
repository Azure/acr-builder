// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"reflect"
	"testing"
)

/* TestResolveMapAndValidate: */
func TestResolveMapAndValidate(t *testing.T) {
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

func TestLoadExternalAlias(t *testing.T) {
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
				[]*string{"nonexistent.yaml"},
				map[string]string{},
				'$',
			},
			Alias{},
		},
		{
			"Single Nonexistent URL",
			true,
			Alias{
				[]string{"https://httpstat.us/404"},
				map[string]string{},
				'$',
			},
			Alias{},
		},
		{
			"Valid Remote",
			false,
			Alias{
				[]string{"https://TODO"},
				map[string]string{},
				'$',
			},
			Alias{
				[]string{"https://TODO"},
				map[string]string{"alias1": "something", "alias1": "something"},
				'$',
			},
		},
		{
			"Valid Files",
			false,
			Alias{
				[]string{"./testdata/preprocessor/valid-external.yaml"},
				map[string]string{},
				'$',
			},
			Alias{
				[]string{"./testdata/preprocessor/valid-external.yaml"},
				map[string]string{"alias1": "something", "alias1": "something"},
				'$',
			},
		},
		{
			"Valid All",
			false,
			Alias{
				[]string{"./testdata/preprocessor/valid-external.yaml", "https://TODO"},
				map[string]string{},
				'$',
			},
			Alias{
				[]string{"./testdata/preprocessor/valid-external.yaml", "https://TODO"},
				map[string]string{"alias1": "something", "alias1": "something"},
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

		eq := reflect.DeepEqual(test.alias, test.expectedAlias)
		if !eq {
			t.Fatalf("Expected output for " + test.name + " differed from actual")
		}
	}
}

func TestAddAliasFromRemote(t *testing.T) {
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
				[]string{"https://TODO,json"},
				map[string]string{},
				'$',
			},
			Alias{},
		},
		{
			"Properly Formatted",
			true,
			Alias{
				[]string{"https://TODO"},
				map[string]string{},
				'$',
			},
			Alias{
				[]string{"https://TODO"},
				map[string]string{"TODO"},
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

func TestAddAliasFromFile(t *testing.T) {
	tests := []struct {
		name        string
		shouldError bool
		alias       Alias
	}{
		{
			"Improperly Formatted",
			true,
			Alias{},
		},
		{
			"Valid Remote",
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
		err := test.alias.resolveMapAndValidate()
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
