package util

import "testing"

func TestStringSequenceEquals(t *testing.T) {
	tests := []struct {
		a        []string
		b        []string
		expected bool
	}{
		{nil, nil, true},
		{[]string{}, nil, false},
		{nil, []string{}, false},
		{[]string{"a"}, []string{"b"}, false},
		{[]string{"a", "b"}, []string{"b"}, false},
		{[]string{"a", "b", "c"}, []string{"a", "b", "c"}, true},
	}
	for _, test := range tests {
		if actual := StringSequenceEquals(test.a, test.b); actual != test.expected {
			t.Errorf("Expected %v and %v to be %v but got %v", test.a, test.b, test.expected, actual)
		}
	}
}

func TestIntSequenceEquals(t *testing.T) {
	tests := []struct {
		a        []int
		b        []int
		expected bool
	}{
		{nil, nil, true},
		{[]int{}, nil, false},
		{nil, []int{}, false},
		{[]int{1}, []int{2}, false},
		{[]int{1, 2}, []int{1}, false},
		{[]int{1, 2, 3}, []int{1, 2, 3}, true},
	}
	for _, test := range tests {
		if actual := IntSequenceEquals(test.a, test.b); actual != test.expected {
			t.Errorf("Expected %v and %v to be %v but got %v", test.a, test.b, test.expected, actual)
		}
	}
}
