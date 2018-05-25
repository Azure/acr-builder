package cli

import "testing"

// TestValidateIsolation_Valid tests valid isolations.
func TestValidateIsolation_Valid(t *testing.T) {
	for k := range isolations {
		if err := validateIsolation(k); err != nil {
			t.Errorf("%s should be a valid isolation but isn't", k)
		}
	}
}

// TestValidateIsolation_Invalid tests invalid isolations.
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
