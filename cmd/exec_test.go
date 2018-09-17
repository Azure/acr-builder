// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package cmd

import (
	"testing"

	"github.com/Azure/acr-builder/templating"
)

func TestNewExecCmd_Defaults(t *testing.T) {
	tests := []struct {
		taskFile         string
		encodedFile      string
		expectedTaskFile string
	}{
		{"", "foo", ""},
		{"", "", defaultTaskFile},
		{"foo.yaml", "", "foo.yaml"},
	}

	for _, test := range tests {
		cmd := &execCmd{}
		cmd.opts = &templating.BaseRenderOptions{
			TaskFile:              test.taskFile,
			Base64EncodedTaskFile: test.encodedFile,
		}
		cmd.setDefaultTaskFile()

		if cmd.opts.TaskFile != test.expectedTaskFile {
			t.Errorf("Expected %s as the task file but got %s", test.expectedTaskFile, cmd.opts.TaskFile)
		}
	}
}
