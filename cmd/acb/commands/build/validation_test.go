// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package build

import "testing"

func TestValidateIsolation_Valid(t *testing.T) {
	for k := range isolations {
		if err := validateIsolation(k); err != nil {
			t.Errorf("%s should be a valid isolation but isn't", k)
		}
	}
}

func TestValidateIsolation_Invalid(t *testing.T) {
	inValidValues := []string{
		"hyperv_isolation",
		"h12",
		"process ",
		"isolation",
	}

	for _, value := range inValidValues {
		if err := validateIsolation(value); err == nil {
			t.Errorf("%s should be an invalid isolation, but it's valid", value)
		}
	}
}

func TestValidatePush_Valid(t *testing.T) {
	if err := validatePush(false, []string{";bar;qux"}); err != nil {
		t.Errorf("Credentials shouldn't be required unless push is specified. Err: %v", err)
	}

	if err := validatePush(true, []string{"foo;bar;qux"}); err != nil {
		t.Errorf("All creds are provided, but received an error: %v", err)
	}
}

func TestValidatePush_Invalid(t *testing.T) {
	for _, test := range []struct {
		push  bool
		creds []string
	}{
		{true, nil},
		{true, []string{}},
	} {
		if err := validatePush(test.push, test.creds); err == nil {
			t.Error("Invalid creds provided but no error was returned")
		}
	}
}
