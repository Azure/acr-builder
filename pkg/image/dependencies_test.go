// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package image

import "testing"

func TestEquals(t *testing.T) {
	var nilImg *Reference

	tests := []struct {
		ref1     *Reference
		ref2     *Reference
		expected bool
	}{
		{
			&Reference{
				Registry: "a",
			},
			nilImg,
			false,
		},
		{
			nilImg,
			&Reference{
				Registry: "b",
			},
			false,
		},
		{
			nilImg,
			nilImg,
			true,
		},
		{
			&Reference{
				Registry: "a",
			},
			&Reference{
				Registry: "a",
			},
			true,
		},
		{
			&Reference{
				Registry: "a",
			},
			&Reference{
				Registry: "b",
			},
			false,
		},
		{
			&Reference{
				Registry:   "a",
				Repository: "b",
				Tag:        "c",
				Digest:     "d",
				Reference:  "e",
			},
			&Reference{
				Registry:   "a",
				Repository: "b",
				Tag:        "c",
				Digest:     "d",
				Reference:  "e",
			},
			true,
		},
		{
			&Reference{
				Registry:   "a",
				Repository: "b",
				Tag:        "c",
				Digest:     "d",
				Reference:  "e",
			},
			&Reference{
				Registry:   "a",
				Repository: "b",
				Tag:        "c",
				Digest:     "d",
				Reference:  "f",
			},
			false,
		},
	}

	for _, test := range tests {
		if actual := Equals(test.ref1, test.ref2); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}

func TestString(t *testing.T) {
	var nilImg *Reference
	tests := []struct {
		ref1     *Reference
		expected string
	}{
		{
			&Reference{
				Registry:   "a",
				Repository: "b",
				Tag:        "c",
				Digest:     "d",
				Reference:  "e",
			},
			"Registry: a\nRepository: b\nTag: c\nDigest: d\nReference: e\n",
		},
		{
			nilImg,
			defaultStringValue,
		},
	}

	for _, test := range tests {
		if actual := test.ref1.String(); actual != test.expected {
			t.Errorf("Expected %v but got %v", test.expected, actual)
		}
	}
}
