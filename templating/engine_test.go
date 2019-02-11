// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"testing"
)

// TestRenderWithEmptySecrets tests whether rendering ignores if no secret vaules are available.
func TestRenderWithEmptySecrets(t *testing.T) {
	base := map[string]interface{}{}
	j := NewTemplate(
		"job1",
		[]byte("{{ .Secrets.mysecret }} - {{ .Values.mykey }}"),
	)
	vals := Values{"mykey": "myvalue"}
	base["Values"] = vals
	base["Secrets"] = Values{}
	expectedMsg := " - myvalue"

	engine := NewEngine()
	rendered, err := engine.Render(j, base)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered)
	}
}

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

func TestRender_RequiredParameters(t *testing.T) {
	engine := NewEngine()
	if _, err := engine.Render(nil, Values{}); err == nil {
		t.Fatal("Expected an error when rendering a nil template")
	}

	if _, err := engine.Render(&Template{}, nil); err == nil {
		t.Fatalf("Expected an error when rendering nil values")
	}
}

// TestRenderMath verifies that the engine can render math sprig funcs.
func TestRenderMath(t *testing.T) {
	expectedMsg := "15,-5,50"
	engine := NewEngine()
	templateName := "math"
	template := NewTemplate(
		templateName,
		[]byte("{{.foo | add .bar}}, {{- .bar | sub .foo }}, {{- .foo | mul .bar}}"),
	)

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
	return NewTemplate(
		name,
		[]byte("{{ .jobName | upper }} - {{ .description | lower }}"),
	)
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
