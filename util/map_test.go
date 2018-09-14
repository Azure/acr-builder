package util

import "testing"

func TestIsMap(t *testing.T) {
	tests := []struct {
		v        interface{}
		expected bool
	}{
		{"", false},
		{map[string]interface{}{}, true},
		{map[string]string{}, false},
	}

	for _, test := range tests {
		if actual := IsMap(test.v); actual != test.expected {
			t.Errorf("Expected %v to be %v but got %v", test.v, test.expected, actual)
		}
	}
}
