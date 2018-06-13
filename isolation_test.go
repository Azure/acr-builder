package main

import "testing"

func TestIsolationValidValues(t *testing.T) {
	validValues := []string{
		"",
		"hyperv",
		"process",
		"default",
	}

	for _, value := range validValues {
		err := validateIsolation(value)

		if err != nil {
			t.Errorf("Expected to be success. But returned error for value %s", value)
		}
	}
}

func TestIsolationInValidValues(t *testing.T) {
	inValidValues := []string{
		"hyperv_isolation",
		"h12",
		"process ",
		"isolation",
	}

	for _, value := range inValidValues {
		err := validateIsolation(value)

		if err == nil {
			t.Errorf("Expected to be failed. But returned success for value %s", value)
		}
	}
}
