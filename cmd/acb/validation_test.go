// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

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

func TestValidateRegistryCreds_Valid(t *testing.T) {
	if err := validateRegistryCreds("", ""); err != nil {
		t.Errorf("No creds passed, but received err: %v", err)
	}

	if err := validateRegistryCreds("foo", "bar"); err != nil {
		t.Errorf("Username/password provided, but returned an err: %v", err)
	}
}

func TestValidateRegistryCreds_Invalid(t *testing.T) {
	if err := validateRegistryCreds("foo", ""); err == nil {
		t.Error("Expected an error from a missing password")
	}

	if err := validateRegistryCreds("", "bar"); err == nil {
		t.Error("Expected an error from a missing username")
	}
}

func TestValidatePush_Valid(t *testing.T) {
	if err := validatePush(false, "", "bar", "qux"); err != nil {
		t.Errorf("Credentials shouldn't be required unless push is specified. Err: %v", err)
	}

	if err := validatePush(true, "foo", "bar", "qux"); err != nil {
		t.Errorf("All creds are provided, but received an error: %v", err)
	}
}

func TestValidatePush_Invalid(t *testing.T) {
	if err := validatePush(true, "", "bar", "qux"); err == nil {
		t.Error("Invalid creds provided but no error was returned")
	}
}
