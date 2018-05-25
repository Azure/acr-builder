package templating

import (
	"testing"
)

// TestRenderAllTemplates tests rendering all templates for a job.
func TestRenderAllTemplates(t *testing.T) {
	jobName := "job1"
	j := makeTestJob(jobName)
	c := makeTestConfig()
	expectedMsg := "FRUITJOB - this is a fruit job"

	engine := New()

	v, err := OverrideValues(j, c)
	if err != nil {
		t.Fatalf("Failed to override values: %v", err)
	}

	rendered, err := engine.RenderAllTemplates(j, v)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered["templates/template1"] != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered["templates/template1"])
	}
}

// TestRenderMath verifies that the engine can render math sprig funcs.
func TestRenderMath(t *testing.T) {
	expectedMsg := "15,-5,50"
	engine := New()
	j := &Job{
		Templates: []*Template{
			{Name: "templates/math", Data: []byte("{{.foo | add .bar}}, {{- .bar | sub .foo }}, {{- .foo | mul .bar}}")},
		},
		Config: &Config{
			RawValue: `
foo = 5
bar = 10
`,
		},
	}

	c := &Config{}

	v, err := OverrideValues(j, c)
	if err != nil {
		t.Fatalf("Failed to override values: %v", err)
	}

	rendered, err := engine.RenderAllTemplates(j, v)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered["templates/math"] != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered["templates/math"])
	}
}

// TestRender_IgnoredTemplate ensures that an ignored template renders as an empty string.
func TestRender_IgnoredTemplate(t *testing.T) {
	jobName := "job1"
	j := makeTestJob(jobName)
	c := makeTestConfig()

	expectedMsg := ""

	engine := New()

	v, err := OverrideValues(j, c)
	if err != nil {
		t.Fatalf("Failed to override values: %v", err)
	}

	// NB: an empty keep set should result in all templates being ignored.
	keep := make(map[string]bool)
	rendered, err := engine.Render(j, v, keep)
	if err != nil {
		t.Errorf("Failed to render. Err: %v", err)
	}

	if rendered["templates/template1"] != expectedMsg {
		t.Errorf("Expected %s, but got %s", expectedMsg, rendered["templates/template1"])
	}
}

func makeTestJob(jobName string) *Job {

	j := &Job{
		Templates: []*Template{
			{Name: "templates/template1", Data: []byte("{{ .jobName | upper }} - {{ .description | lower }}")},
		},
		Config: &Config{
			RawValue: `
jobName = "fruitjob"
description = "this should get overridden"
`,
		},
	}

	return j
}

func makeTestConfig() *Config {
	c := &Config{
		RawValue: `
description = "THIS IS a FRuiT JoB"
`,
	}

	return c
}
