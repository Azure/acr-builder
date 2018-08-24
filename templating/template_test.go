// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import "testing"

func TestNewTemplate(t *testing.T) {
	tests := []struct {
		name             string
		template         string
		expectedTemplate string
	}{
		{"", "", ""},
		{"a", "", ""},
		{"b", "# Something with a comment", ""},

		{"c",
			`# comments
steps:
 # should be skipped
  - build: -f Dockerfile -t . # this should be preserved
           #everywhere
  #even at the end of the file`,

			`steps:
  - build: -f Dockerfile -t . # this should be preserved`},
	}

	for _, test := range tests {
		template := NewTemplate(test.name, []byte(test.template))
		if template.Name != test.name {
			t.Errorf("Expected %s as the template name, but got %s", test.name, template.Name)
		}
		sData := string(template.Data)
		if sData != test.expectedTemplate {
			t.Errorf("Expected \n'%s'\n as template data, but got \n'%s'\n", test.expectedTemplate, sData)
		}
	}
}
