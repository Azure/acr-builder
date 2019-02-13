// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package templating

import (
	"context"
	"testing"
	"time"
)

func TestParseValues_Valid(t *testing.T) {
	tests := []struct {
		values   []string
		expected string
	}{
		{
			[]string{"a=b", "b===ll", "c=12345", "d=ab=", "e=", "f=sadf=234"},
			`a: b
b: ==ll
c: "12345"
d: ab=
e: ""
f: sadf=234
`,
		},
		{
			[]string{"a=b", "a=c", "a=d"},
			`a: d
`,
		},
	}

	for _, test := range tests {
		actual, err := parseValues(test.values)
		if err != nil {
			t.Errorf("Failed to parse vals, err: %v", err)
		}
		if actual != test.expected {
			t.Errorf("Failed to parse values, expected '%s' but got '%s'", test.expected, actual)
		}
	}
}

func TestParseValues_Invalid(t *testing.T) {
	tests := []struct {
		values []string
	}{
		{[]string{"apple", "=k", "=====", "=", "", "           "}},
	}

	for _, test := range tests {
		if _, err := parseValues(test.values); err == nil {
			t.Errorf("Expected an error during parse values, but it was nil")
		}
	}
}

func TestParseRegistryName(t *testing.T) {
	tests := []struct {
		fullyQualifiedRegistryName string
		expectedRegistryName       string
	}{
		{"", ""},
		{"foo", "foo"},
		{"foo.azurecr.io", "foo"},
		{"foo-bar.azurecr-test.io", "foo-bar"},
		{"  ", "  "},
	}

	for _, test := range tests {
		if actual := parseRegistryName(test.fullyQualifiedRegistryName); actual != test.expectedRegistryName {
			t.Errorf("Expected %s but got %s for the registry name", test.expectedRegistryName, actual)
		}
	}
}

func TestLoadAndRenderSteps(t *testing.T) {
	opts := &BaseRenderOptions{
		TaskFile:   "testdata/caching/cache.yaml",
		ValuesFile: "testdata/caching/values.yaml",
	}
	expected := `steps:
  - id: "puller"
    cmd: docker pull golang:1.10.1-stretch

  - id: build-foo
    cmd: build -f Dockerfile https://github.com/Azure/acr-builder.git --cache-from=ubuntu

  - id: build-bar
    cmd: build -f Dockerfile https://github.com/Azure/acr-builder.git --cache-from=ubuntu
    when: ["puller"]`

	var template *Template
	template, err := LoadTemplate(opts.TaskFile)
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}

	actual, err := LoadAndRenderSteps(context.Background(), template, opts)
	if err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
	expected = adjustCRInExpectedStringOnWindows(expected)
	if actual != expected {
		t.Errorf("Expected \n%s\n but got \n%s\n", expected, actual)
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		original string
		expected string
	}{
		{"", "''"},
		{`''`, `''"'"''"'"''`},
		{`'single quotes'`, `''"'"'single quotes'"'"''`},
		{"double quotes", "'double quotes'"},
		{`no quotes`, `'no quotes'`},
		{`{;$\}`, `'{;$\}'`},
		{`nothingtoescape`, `nothingtoescape`},
		{`{"val": "foo", "bar": "something#@$!()"}`, `'{"val": "foo", "bar": "something#@$!()"}'`},
	}

	for _, test := range tests {
		if actual := shellQuote(test.original); actual != test.expected {
			t.Fatalf("Expected %s but got %s", test.expected, actual)
		}
	}
}

// TestRenderAndResolveSecrets tests rendering templates with secrets, resolve and adding Secrets.
func TestRenderAndResolveSecrets(t *testing.T) {
	renderOpts := &BaseRenderOptions{
		SecretResolveTimeout: time.Minute * 5,
	}

	ctx := context.Background()

	tests := []struct {
		template     string
		vaulesMap    Values
		secretValues Values
	}{
		{`
secrets:
  - id: mysecret
    akv: https://myvault.vault.azure.net/secrets/mysecret
  - id: mysecret1
    akv: https://myvault.vault.azure.net/secrets/mysecret1
    clientID: c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86`,
			Values{"mykey": "myvalue"},
			Values{"mysecret": "https://myvault.vault.azure.net/secrets/mysecret-", "mysecret1": "https://myvault.vault.azure.net/secrets/mysecret1-c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
		},
		// Simple templating in secrets section
		{`
secrets:
  - id: mysecret
    akv: {{.Values.myakv}}
  - id: {{.Values.myid2}}
    akv: {{.Values.myakv2}}
    clientID: {{.Values.myclientID}}`,
			Values{"myid2": "mysecret1", "myakv": "https://myvault.vault.azure.net/secrets/mysecret", "myakv2": "https://myvault.vault.azure.net/secrets/mysecret1", "myclientID": "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
			Values{"mysecret": "https://myvault.vault.azure.net/secrets/mysecret-", "mysecret1": "https://myvault.vault.azure.net/secrets/mysecret1-c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
		},
		{`
secrets:
  - id: mysecret
    akv: {{.Values.myakv}}
  - id: {{.Run.ID}}_{{.Values.myid2}}
    akv: {{.Values.myakv2}}
    clientID: {{.Values.myclientID}}`,
			Values{"ID": "runId", "myid2": "mysecret1", "myakv": "https://myvault.vault.azure.net/secrets/mysecret", "myakv2": "https://myvault.vault.azure.net/secrets/mysecret1", "myclientID": "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
			Values{"mysecret": "https://myvault.vault.azure.net/secrets/mysecret-", "runId_mysecret1": "https://myvault.vault.azure.net/secrets/mysecret1-c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86"},
		},
		{`
secrets:
  - id: mysecret
    akv: myakv`,
			Values{"mykey": "myvalue"},
			Values{"mysecret": "myakv-"},
		},
		{`
steps:
  - id: secrets
    cmd: bash echo hello world`,
			Values{"mykey": "myvalue"},
			Values{},
		},
	}

	for _, test := range tests {
		base := Values{}
		base["Values"] = test.vaulesMap
		base["Run"] = test.vaulesMap
		template := NewTemplate(
			"job1",
			[]byte(test.template),
		)
		engine := NewEngine()
		secrets, err := renderAndResolveSecrets(ctx, template, engine, MockResolveSecret, renderOpts, base)
		if err != nil {
			t.Errorf("Test failed with error %v", err)
		}

		if test.secretValues == nil {
			if secrets != nil {
				t.Errorf("Secrets do not match. Expected  %v but got %v", test.secretValues, secrets)
			}
		} else {
			if len(secrets) != len(test.secretValues) {
				t.Errorf("Expected number of secrets: %v, but got %v", len(test.secretValues), len(secrets))
			}
			for key, value := range test.secretValues {
				if secrets[key] != value {
					t.Errorf("Secrets donot match. Expected  %v but got %v", test.secretValues, secrets)
				}
			}
		}

	}
}
