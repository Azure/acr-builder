// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"testing"
	"text/template"
)

const (
	curiePath     = "testdata/curie/curie.toml"
	curieValsPath = "testdata/curie/values.toml"

	eCurieFirst    = "Marie"
	eCurieLast     = "Curie"
	eCurieBorn     = "1867"
	eCurieResearch = "radioactivity"
	eCurieFrom     = "Poland"
	eCurieAwards   = "[map[id:Nobel Prize in Physics] map[id:Davy Medal] map[id:Albert Medal]]"
)

// TestDeserialize tests deserialization of bytes to Values.
func TestDeserialize(t *testing.T) {
	data := `# TestDeserialize
title = "A grocery list"
fruits = ["banana", "apple", "pear"]

[fruit]
  [fruit.fruit]
	nested = "star"
	fruit = "nested"
`
	v, err := Deserialize([]byte(data))
	if err != nil {
		t.Fatalf("Error deserializing: %v", err)
	}

	matchFruits(t, v)
}

// TestDeserializeFromFile tests deserialization of a file to Values.
func TestDeserializeFromFile(t *testing.T) {
	v, err := DeserializeFromFile("./testdata/fruits.toml")
	if err != nil {
		t.Fatalf("Failed to read file. Err: %v", err)
	}

	matchFruits(t, v)
}

// TestToTOMLString converts a string to TOML.
func TestToTOMLString(t *testing.T) {
	vals := Values{"id": true, "hello": 1, "someString": "something"}
	expected := `hello = 1
id = true
someString = "something"
`
	out, err := vals.ToTOMLString()
	if err != nil {
		t.Fatalf("Failed to convert to TOML string. Err: %v", err)
	}

	if expected != out {
		t.Fatalf("Expected %s but got %s", expected, out)
	}
}

// TestOverrideValues ensures that values.toml overrides the default data successfully.
func TestOverrideValues(t *testing.T) {
	c1, err := LoadConfig(curiePath)
	if err != nil {
		t.Fatal(err)
	}

	c2, err := LoadConfig(curieValsPath)
	if err != nil {
		t.Fatal(err)
	}

	vals, err := OverrideValues(c1, c2)
	if err != nil {
		t.Fatal(err)
	}

	tests := []renderable{
		{"{{.born}}", eCurieBorn},
		{"{{.first}}", eCurieFirst},
		{"{{ .last }}", eCurieLast},
		{"{{.research}}", eCurieResearch},
		{"{{.from}}", eCurieFrom},
		{"{{.awards}}", eCurieAwards},
	}

	for _, test := range tests {
		if o, err := executeTemplate(test.tpl, vals); err != nil || o != test.expect {
			t.Errorf("Expected %q to expand to %q. Received %q", test.tpl, test.expect, o)
		}
	}
}

// TestOverrideValuesWithBuildInfo tests that a job gets overridden with base properties
// and maintains its original values.
func TestOverrideValuesWithBuildInfo(t *testing.T) {
	c1, err := LoadConfig(curiePath)
	if err != nil {
		t.Fatal(err)
	}

	c2, err := LoadConfig(curieValsPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedID := "SomeID"
	expectedCommit := "Some Commit"
	expectedTag := "some Tag"
	expectedRepo := "some RePo"
	expectedBranch := "br"
	expectedTrigger := "triggered from someone cool!!1"

	options := BaseRenderOptions{
		ID:          expectedID,
		Commit:      expectedCommit,
		Tag:         expectedTag,
		Repository:  expectedRepo,
		Branch:      expectedBranch,
		TriggeredBy: expectedTrigger,
	}
	vals, err := OverrideValuesWithBuildInfo(c1, c2, options)
	if err != nil {
		t.Fatal(err)
	}

	tests := []renderable{
		// Base properties
		{"{{.Build.ID}}", expectedID},
		{"{{.Build.Commit}}", expectedCommit},
		{"{{ .Build.Tag }}", expectedTag},
		{"{{ .Build.Repository}}", expectedRepo},
		{"{{.Build.Branch}}", expectedBranch},
		{"{{.Build.TriggeredBy}}", expectedTrigger},
		{"{{.Values.born}}", eCurieBorn},
		{"{{.Values.first}}", eCurieFirst},
		{"{{.Values.last}}", eCurieLast},
		{"{{.Values.from}}", eCurieFrom},
		{"{{.Values.awards}}", eCurieAwards},
	}
	for _, test := range tests {
		if o, err := executeTemplate(test.tpl, vals); err != nil || o != test.expect {
			t.Errorf("Expected %q to expand to %q. Received %q", test.tpl, test.expect, o)
		}
	}
}

func matchFruits(t *testing.T, m map[string]interface{}) {
	if m["title"] != "A grocery list" {
		t.Errorf("Unexpected title: %s", m["title"])
	}

	if o, err := executeTemplate("{{len .fruits}}", m); err != nil {
		t.Errorf("# of fruits: %s", err)
	} else if o != "3" {
		t.Errorf("Expected 3 fruits, but got %s", o)
	}

	if o, err := executeTemplate("{{.fruit.fruit.nested}}", m); err != nil {
		t.Errorf(".fruit.fruit.nested: %s", err)
	} else if o != "star" {
		t.Errorf("Expected nested fruit to be star")
	}
}

func executeTemplate(ts string, v map[string]interface{}) (string, error) {
	var b bytes.Buffer
	t := template.Must(template.New("_").Parse(ts))
	if err := t.Execute(&b, v); err != nil {
		return "", err
	}
	return b.String(), nil
}

type renderable struct {
	tpl    string
	expect string
}
