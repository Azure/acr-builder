// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"bytes"
	"reflect"
	"testing"
	"text/template"
	"time"
)

const (
	curiePath     = "testdata/curie/curie.yaml"
	curieValsPath = "testdata/curie/values.yaml"

	eCurieFirst    = "Marie"
	eCurieLast     = "Curie"
	eCurieBorn     = "1867"
	eCurieResearch = "radioactivity"
	eCurieFrom     = "Poland"
	eCurieAwards   = "[Nobel Prize in Physics Davy Medal Albert Medal]"
)

// TestDeserialize tests deserialization of bytes to Values.
func TestDeserialize(t *testing.T) {
	data := `# TestDeserialize
title: A grocery list
fruits: [banana, apple, pear]
fruit:
  fruit:
    nested: star
    fruit: nested
`
	v, err := Deserialize([]byte(data))
	if err != nil {
		t.Fatalf("Error deserializing: %v", err)
	}

	matchFruits(t, v)
}

// TestDeserializeFromFile tests deserialization of a file to Values.
func TestDeserializeFromFile(t *testing.T) {
	v, err := DeserializeFromFile("./testdata/fruits.yaml")
	if err != nil {
		t.Fatalf("Failed to read file. Err: %v", err)
	}

	matchFruits(t, v)
}

func TestDeserializeFromFile_Defaults(t *testing.T) {
	_, err := DeserializeFromFile("")
	if err == nil {
		t.Fatal("Expected to receive an error from an invalid file name")
	}
}

// TestToYAMLString converts a string to YAML.
func TestToYAMLString(t *testing.T) {
	vals := Values{"id": true, "hello": 1, "someString": "something"}
	expected := `hello: 1
id: true
someString: something
`
	out, err := vals.ToYAMLString()
	if err != nil {
		t.Fatalf("Failed to convert to YAML string. Err: %v", err)
	}

	if expected != out {
		t.Fatalf("Expected %s but got %s", expected, out)
	}
}

// TestOverrideValues ensures that values file overrides the default data successfully.
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
	expectedRepository := "some RePo"
	expectedBranch := "br"
	expectedTriggeredBy := "triggered from someone cool!!1"
	expectedRegistry := "foo.azurecr.io"
	expectedRegistryName := "foo"
	expectedGitTag := "some git tag"
	expectedSharedVolume := "acb_home_vol_12345"
	expectedOS := "linux"
	expectedOSVersion := "1903"
	expectedArchitecture := "amd64"

	parsedTime, err := time.Parse("20060102-150405", "20100520-131422")
	if err != nil {
		t.Fatal(err)
	}

	expectedTime := "20100520-131422z"

	options := &BaseRenderOptions{
		ID:           expectedID,
		Commit:       expectedCommit,
		Repository:   expectedRepository,
		Branch:       expectedBranch,
		TriggeredBy:  expectedTriggeredBy,
		Registry:     expectedRegistry,
		GitTag:       expectedGitTag,
		Date:         parsedTime,
		SharedVolume: expectedSharedVolume,
		OS:           expectedOS,
		OSVersion:    expectedOSVersion,
		Architecture: expectedArchitecture,
	}
	vals, err := OverrideValuesWithBuildInfo(c1, c2, options)
	if err != nil {
		t.Fatal(err)
	}

	tests := []renderable{
		// Base properties
		{"{{.Build.ID}}", expectedID},
		{"{{.Run.ID}}", expectedID},
		{"{{.Run.Commit}}", expectedCommit},
		{"{{ .Run.Repository}}", expectedRepository},
		{"{{.Run.Branch}}", expectedBranch},
		{"{{.Run.TriggeredBy}}", expectedTriggeredBy},
		{"{{.Run.Registry}}", expectedRegistry},
		{"{{.Run.RegistryName}}", expectedRegistryName},
		{"{{.Run.GitTag}}", expectedGitTag},
		{"{{.Run.Date}}", expectedTime},
		{"{{.Run.SharedVolume}}", expectedSharedVolume},
		{"{{.Run.OS}}", expectedOS},
		{"{{.Run.OSVersion}}", expectedOSVersion},
		{"{{.Run.Architecture}}", expectedArchitecture},
		{"{{.Values.born}}", eCurieBorn},
		{"{{.Values.first}}", eCurieFirst},
		{"{{.Values.last}}", eCurieLast},
		{"{{.Values.from}}", eCurieFrom},
		{"{{.Values.awards }}", eCurieAwards},
		{"{{.ValuesJSON}}", "'{\"awards\":[\"Nobel Prize in Physics\",\"Davy Medal\",\"Albert Medal\"]," +
			"\"born\":" + eCurieBorn +
			",\"first\":\"" + eCurieFirst +
			"\",\"from\":\"" + eCurieFrom +
			"\",\"last\":\"" + eCurieLast +
			"\",\"research\":\"" + eCurieResearch + "\"}'"},
		{"{{.RunJSON}}", "'{\"Architecture\":\"" + expectedArchitecture +
			"\",\"Branch\":\"" + expectedBranch +
			"\",\"Commit\":\"" + expectedCommit +
			"\",\"Date\":\"" + expectedTime +
			"\",\"GitTag\":\"" + expectedGitTag +
			"\",\"ID\":\"" + expectedID +
			"\",\"OS\":\"" + expectedOS +
			"\",\"OSVersion\":\"" + expectedOSVersion +
			"\",\"Registry\":\"" + expectedRegistry +
			"\",\"RegistryName\":\"" + expectedRegistryName +
			"\",\"Repository\":\"" + expectedRepository +
			"\",\"SharedVolume\":\"" + expectedSharedVolume +
			"\",\"TriggeredBy\":\"" + expectedTriggeredBy + "\"}'"},
	}
	for _, test := range tests {
		if o, err := executeTemplate(test.tpl, vals); err != nil || o != test.expect {
			t.Errorf("Expected %q to expand to %q. Received %q", test.tpl, test.expect, o)
		}
	}
}

func TestMergeMaps(t *testing.T) {
	sink := make(map[string]interface{})
	nestedMap := make(map[string]interface{})
	nestedMap["innerMap"] = "innerValue"
	nestedMap["innerString"] = "innerStringValue"
	nestedMap["innerInt"] = 30

	sink["foo"] = 3
	sink["bar"] = ""
	sink["nested"] = nestedMap

	expectedFooValue := 5
	expectedBarValue := "qux"

	source := make(map[string]interface{})
	source["foo"] = expectedFooValue
	sink["bar"] = expectedBarValue

	expected := make(map[string]interface{})
	expected["foo"] = expectedFooValue
	expected["bar"] = expectedBarValue
	expected["nested"] = nestedMap

	actual := mergeMaps(sink, source)

	if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Expected %v but got %v", expected, actual)
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
