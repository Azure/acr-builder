// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"testing"
)

// TestRenderAllTemplates tests rendering all templates for a job.
func TestRenderAllTemplates(t *testing.T) {
	jobName := "job1"
	j := makeTestTemplate(jobName)
	c1 := makeDefaultTestConfig()
	c2 := makeTestConfig()
	expectedMsg := "FRUITJOB - this is a fruit job"

	engine := NewEngine()

	v, err := OverrideValues(c1, c2)
	if err != nil {
		t.Fatalf("Failed to override values: %v", err)
	}

	rendered, err := engine.Render(j, v)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered)
	}
}

// TestRenderMath verifies that the engine can render math sprig funcs.
func TestRenderMath(t *testing.T) {
	expectedMsg := "15,-5,50"
	engine := NewEngine()
	templateName := "math"
	template := &Template{
		Name: templateName,
		Data: []byte("{{.foo | add .bar}}, {{- .bar | sub .foo }}, {{- .foo | mul .bar}}"),
	}

	c1 := &Config{
		RawValue: `
foo: 5
bar: 10
`,
	}
	c2 := &Config{}

	v, err := OverrideValues(c1, c2)
	if err != nil {
		t.Fatalf("Failed to override values: %v", err)
	}

	rendered, err := engine.Render(template, v)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered)
	}
}

func makeTestTemplate(name string) *Template {
	return &Template{
		Name: name,
		Data: []byte("{{ .jobName | upper }} - {{ .description | lower }}"),
	}
}

func makeDefaultTestConfig() *Config {
	return &Config{
		RawValue: `
description: this will be overridden
jobName: this will be overridden
`,
	}
}

func makeTestConfig() *Config {
	return &Config{
		RawValue: `
description: THIS IS a FRuiT JoB
jobName: FRUITJOB
`,
	}
}
