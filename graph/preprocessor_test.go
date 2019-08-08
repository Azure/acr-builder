// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"context"
	"testing"

	"github.com/Azure/acr-builder/pkg/procmanager"
	"github.com/Azure/acr-builder/util"
)

/* TestResolveMapAndValidate: */
func TestResolveMapAndValidate(t *testing.T) {
	tests := []struct {
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Improper Directive Choice",
			true,
			Alias{}
		},
		{
			"Improper Key Name",
			true,
			Alias{}
		},
		{
			"Improper Directive Length",
			true,
			Alias{}
		},
		{
			"Valid Alias",
			false,
			Alias{}
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
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Single Nonexistent File",
			true,
			Alias{}
		},
		{
			"Single Nonexistent URL",
			true,
			Alias{}
		},
		{
			"Valid Remote",
			false,
			Alias{}
		},
		{
			"Valid Files",
			false,
			Alias{}
		},
		{
			"Valid All",
			false,
			Alias{}
		}
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
func TestAddAliasFromRemote(t *testing.T) {
	tests := []struct {
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Improperly Formatted",
			true,
			Alias{}
		},
		{
			"Valid Remote",
			true,
			Alias{}
		}
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
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Improperly Formatted",
			true,
			Alias{}
		},
		{
			"Valid Remote",
			true,
			Alias{}
		}
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
		name               string
		shouldError        bool
		isFile bool
		value string
		
	}{
		{
			"Data from ACR Task Json",
			false,
			true,
			"somefilename"
		},
		{
			"Data from ACR Task Commandline String",
			false,
			false,
			"somefilename"
		},
		{
			"Invalid Task from File",
			true,
			true,
			"somefilename"
		},
		{
			"Invalid Commandline String",
			true,
			false,
			"somefilename"
		},
		{
			"Nested Values",
			true,
			true,
			"somefilename"
		}
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
		name               string
		shouldError        bool
		isFile bool
		value string
		
	}{
		{
			"Chained values",
			false,
			true,
			"somefilename"
		},
		{
			"Chained values changed directive",
			false,
			true,
			"somefilename"
		},
		{
			"Multiline replacements",
			true,
			true,
			"somefilename"
		},
		{
			"Complex command",
			true,
			false,
			"somecommand"
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
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Improper Directive Choice",
			true,
			Alias{}
		},
		{
			"Improper Key Name",
			true,
			Alias{}
		},
		{
			"Improper Directive Length",
			true,
			Alias{}
		},
		{
			"Valid Alias",
			false,
			Alias{}
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
		name               string
		shouldError        bool
		alias Alias
	}{
		{
			"Proper Pass File",
			false,
			Alias{}
		},
		{
			"Proper Pass Commands",
			false,
			Alias{}
		},
		{
			"Proper Pass External definitions",
			false,
			Alias{}
		}
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

