// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"runtime"
	"strings"
	"testing"
)

const (
	thamesValuesPath = "testdata/thames/values.yaml"
	thamesPath       = "testdata/thames/thames.yaml"

	expectedConfig = `# Default values for the Thames job
name: Thames
country: England
counties: [Gloucestershire, Wiltshire, Oxfordshire, Berkshire, Buckinghamshire, Surrey]
length: 346
elevation: 0
apiName: v1`

	expectedTemplate = `apiName: "{{.Values.apiName}}"

metadata:
  - buildId: "{{.Run.ID | upper}}"
    commit: "{{.Run.Commit | lower}}"
    tag: "{{.Run.Tag}}"
    repository: "{{.Run.Repository}}"
    branch: "{{.Run.Branch}}"
    triggeredBy: "{{.Run.TriggeredBy}}"`
)

func TestLoadConfig(t *testing.T) {
	c, err := LoadConfig(thamesValuesPath)
	if err != nil {
		t.Fatalf("failed to load config from path %s. Err: %v", thamesValuesPath, err)
	}
	if c == nil {
		t.Fatalf("resulting config at path %s was nil", thamesValuesPath)
	}
	actual := c.GetRawValue()
	if adjustCRInExpectedStringOnWindows(expectedConfig) != actual {
		t.Fatalf("expected %s but got %s", expectedConfig, actual)
	}
}

func TestLoadConfig_Invalid(t *testing.T) {
	_, err := LoadConfig("")
	if err == nil {
		t.Fatal("Expected load config to fail because of an invalid path")
	}
}

func TestDecodeConfig(t *testing.T) {
	enc := "IyBEZWZhdWx0IHZhbHVlcyBmb3IgdGhlIFRoYW1lcyBqb2IKbmFtZTogVGhhbWVzCmNvdW50cnk6IEVuZ2xhbmQKY291bnRpZXM6IFtHbG91Y2VzdGVyc2hpcmUsIFdpbHRzaGlyZSwgT3hmb3Jkc2hpcmUsIEJlcmtzaGlyZSwgQnVja2luZ2hhbXNoaXJlLCBTdXJyZXldCmxlbmd0aDogMzQ2CmVsZXZhdGlvbjogMAphcGlOYW1lOiB2MQ=="
	c, err := DecodeConfig(enc)
	if err != nil {
		t.Fatalf("failed to decode config, err: %v", err)
	}
	if c == nil {
		t.Fatal("resulting config was nil")
	}
	actual := c.GetRawValue()
	if expectedConfig != actual {
		t.Fatalf("expected %s but got %s", expectedConfig, actual)
	}
}

func TestDecodeConfig_Invalid(t *testing.T) {
	enc := "invalidencoding"
	_, err := DecodeConfig(enc)
	if err == nil {
		t.Fatal("expected to fail the decoding with an invalid base64 encoding")
	}
}

func TestLoadTemplate(t *testing.T) {
	template, err := LoadTemplate(thamesPath)
	if err != nil {
		t.Fatalf("failed to load config from path %s. Err: %v", thamesPath, err)
	}
	if template == nil {
		t.Fatalf("resulting template at path %s was nil", thamesPath)
	}
	actual := string(template.GetData())
	if adjustCRInExpectedStringOnWindows(expectedTemplate) != actual {
		t.Fatalf("expected \n'%s'\n as the data but got \n'%s'\n", expectedTemplate, actual)
	}
	expectedName := thamesPath
	if expectedName != template.GetName() {
		t.Fatalf("expected %s as the template's name but got %s", expectedName, template.GetName())
	}
}

func TestLoadTemplate_Invalid(t *testing.T) {
	_, err := LoadTemplate("")
	if err == nil {
		t.Fatal("expected to fail template load because of an invalid file path")
	}
}

func TestDecodeTemplate(t *testing.T) {
	enc := "YXBpTmFtZTogInt7LlZhbHVlcy5hcGlOYW1lfX0iCgptZXRhZGF0YToKICAtIGJ1aWxkSWQ6ICJ7ey5SdW4uSUQgfCB1cHBlcn19IgogICAgY29tbWl0OiAie3suUnVuLkNvbW1pdCB8IGxvd2VyfX0iCiAgICB0YWc6ICJ7ey5SdW4uVGFnfX0iCiAgICByZXBvc2l0b3J5OiAie3suUnVuLlJlcG9zaXRvcnl9fSIKICAgIGJyYW5jaDogInt7LlJ1bi5CcmFuY2h9fSIKICAgIHRyaWdnZXJlZEJ5OiAie3suUnVuLlRyaWdnZXJlZEJ5fX0i"
	template, err := DecodeTemplate(enc)
	if err != nil {
		t.Fatalf("failed to decode template, err: %v", err)
	}
	if template == nil {
		t.Fatal("resulting template was nil")
	}
	actual := string(template.GetData())
	if expectedTemplate != actual {
		t.Fatalf("expected %s as the data but got %s", expectedTemplate, actual)
	}
	expectedName := decodedTemplateName
	if expectedName != template.GetName() {
		t.Fatalf("expected %s as the template's name but got %s", expectedName, template.GetName())
	}
}

func TestDecodeTemplate_Invalid(t *testing.T) {
	enc := "invalidbase64encoding"
	_, err := DecodeTemplate(enc)
	if err == nil {
		t.Fatal("expected to fail template decoding because of an invalid base64 encoding")
	}
}

func adjustCRInExpectedStringOnWindows(expectedStr string) string{
	if runtime.GOOS == "windows" {
		return strings.Replace(expectedStr, "\n", "\r\n", -1)
	}
	return expectedStr
}