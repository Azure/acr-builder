// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package procmanager

import (
	"bytes"
	"context"
	"testing"
)

func TestDryRun(t *testing.T) {
	pm := NewProcManager(true)
	if err := pm.Run(context.Background(), nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
}

func TestRun_NilArgs(t *testing.T) {
	pm := NewProcManager(false)
	if err := pm.Run(context.Background(), nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
}

func TestContainsAnyError(t *testing.T) {
	tests := []struct {
		errors    []string
		stdOutBuf *bytes.Buffer
		stdErrBuf *bytes.Buffer
		expected  bool
	}{
		{
			[]string{"error1", "error2"},
			new(bytes.Buffer),
			new(bytes.Buffer),
			false,
		},
		{
			[]string{"error1", "error2"},
			bytes.NewBufferString("abc error1"),
			bytes.NewBufferString(""),
			true,
		},
		{
			[]string{"error1", "error2"},
			bytes.NewBufferString("abc"),
			bytes.NewBufferString("error2"),
			true,
		},
		{
			[]string{"error1", "error2"},
			bytes.NewBufferString("abc"),
			bytes.NewBufferString("def"),
			false,
		},
	}

	for _, test := range tests {
		if actual := containsAnyError(test.errors, test.stdErrBuf, test.stdOutBuf); actual != test.expected {
			t.Errorf("expected %t but got %t", test.expected, actual)
		}
	}
}
