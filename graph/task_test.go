package graph

import (
	"testing"
)

func TestUsingRegistryCreds(t *testing.T) {
	tests := []struct {
		registry string
		user     string
		pw       string
		cred     string
		expected bool
	}{
		{"foo.azurecr.io", "user", "pw", `{"type": "opaque", "registry": "foo.azurecr.io", "username": "user", "password": "pw", "identity": ""}`, true},
		{"foo.azurecr.io", "user", "", `{"type": "opaque", "registry": "foo.azurecr.io", "username": "user", "password": "", "identity": ""}`, false},
		{"foo.azurecr.io", "", "pw", `{"type": "opaque", "registry": "foo.azurecr.io", "username": "", "password": "pw", "identity": ""}`, false},
		{"", "user", "pw", `{"type": "opaque", "registry": "", "username": "user", "password": "pw", "identity": ""}`, false},
		{"", "user", "", `{"type": "opaque", "registry": "", "username": "user", "password": "pw", "identity": ""}`, false},
		{"", "", "pw", `{"type": "opaque", "registry": "", "username": "", "password": "pw", "identity": ""}`, false},
		{"", "", "", `{"type": "opaque", "registry": "", "username": "", "password": "", "identity": ""}`, false},
	}

	for _, test := range tests {
		cred, err := CreateRegistryCredentialFromString(test.cred)
		if !test.expected {
			if err == nil {
				t.Fatalf("Expected to error out, but did not: %v", test)
			}
			continue
		}

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}

		task := &Task{
			RegistryName: test.registry,
			Credentials:  []*RegistryCredential{cred},
		}
		actual := task.UsingRegistryCreds()
		if test.expected != actual {
			t.Errorf("expected use of registry creds to be %v but got %v", test.expected, actual)
		}
	}
}

func TestNewTask(t *testing.T) {
	tests := []struct {
<<<<<<< HEAD
		steps           []*Step
		secrets         []*Secret
		registry        string
		username        string
		password        string
		okCredentials   bool
		isBuildTask     bool
		expectedVersion string
=======
		steps                []*Step
		secrets              []*Secret
		registry             string
		username             string
		password             string
		credentialString     string
		okCredentials        bool
		isBuildTask          bool
		totalTimeout         int
		expectedTotalTimeout int
		expectedVersion      string
>>>>>>> setup registry credential to support vault/msi (part-1)
	}{
		{nil, nil, "registry", "username", "password", true, true, currentTaskVersion},
		{[]*Step{}, []*Secret{}, "", "", "", false, false, currentTaskVersion},
	}

	for _, test := range tests {
		cred, err := CreateRegistryCredentialFromString(test.credentialString)

		if !test.okCredentials {
			if err == nil {
				t.Fatalf("Expected to error out, but did not: %v", test)
			}
		}

<<<<<<< HEAD
		task, err := NewTask(test.steps, test.secrets, test.registry, []*Credential{cred}, test.isBuildTask)
=======
		task, err := NewTask(test.steps, test.secrets, test.registry, []*RegistryCredential{cred}, test.totalTimeout, test.isBuildTask)
>>>>>>> setup registry credential to support vault/msi (part-1)
		if err != nil {
			t.Fatalf("Unexpected err while creating task: %v", err)
		}
		if task.Version != test.expectedVersion {
			t.Fatalf("expected version: %s but got %s", test.expectedVersion, task.Version)
		}
		actualNumSteps := len(task.Steps)
		expectedNumSteps := len(test.steps)
		if actualNumSteps != expectedNumSteps {
			t.Fatalf("Expected %v steps but got %v", expectedNumSteps, actualNumSteps)
		}
		for i := 0; i < actualNumSteps; i++ {
			if !task.Steps[i].Equals(test.steps[i]) {
				t.Fatalf("Step didn't match, got %v, expected %v", task.Steps[i], test.steps[i])
			}
		}
		if task.RegistryName != test.registry {
			t.Fatalf("Expected %v as the registry but got %v", test.registry, task.RegistryName)
		}
		if test.username != "" && task.Credentials[0].Username != test.username {
			t.Fatalf("Expected %v as the registry username but got %v", test.username, task.Credentials[0].Username)
		}
		if test.password != "" && task.Credentials[0].Password != test.password {
			t.Fatalf("Expected %v as the registry password but got %v", test.password, task.Credentials[0].Password)
		}
		if task.IsBuildTask != test.isBuildTask {
			t.Fatalf("Expected %v as build task but got %v", test.isBuildTask, task.IsBuildTask)
		}
	}
}

func TestInitializeTimeouts(t *testing.T) {
	tests := []struct {
		steps               []*Step
		stepTimeout         int
		expectedStepTimeout int
	}{
		{nil, 0, defaultStepTimeoutInSeconds},
		{nil, 20000, 20000},
	}

	for _, test := range tests {
		task := &Task{
			Steps:       test.steps,
			StepTimeout: test.stepTimeout,
		}
		err := task.initialize()
		if err != nil {
			t.Fatalf("Unexpected err during initialization: %v", err)
		}

		if task.StepTimeout != test.expectedStepTimeout {
			t.Fatalf("Expected %v as the step timeout but got %v", test.expectedStepTimeout, task.StepTimeout)
		}
	}
}

func TestMergingEnvs(t *testing.T) {
	stepEnvsTests := [][]string{
		{},
		{"key1=newVal1", "key2=newVal2"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3="},
		{},
		{"key1=newVal1", "key2=newVal2"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3="},
	}
	taskEnvsTests := [][]string{
		{"key1=val1", "key2=val2", "key3=val3"},
		{"key1=val1", "key2=val2", "key3=val3"},
		{"key1=val1,key2=val2,key3=val3"},
		{"key1=val1,key2=val2,key3=val3"},
		{"key1=val1,key2=val2", "key3=val3,key4=val4"},
		{"key1=val1,key2=val2", "key3=val3,key4=val4"},
	}

	// stepEnvs should overwrite envs that exist in taskEnvs
	expects := [][]string{
		{"key1=val1", "key2=val2", "key3=val3"},
		{"key1=newVal1", "key2=newVal2", "key3=val3"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3="},
		{"key1=val1", "key2=val2", "key3=val3"},
		{"key1=newVal1", "key2=newVal2", "key3=val3", "key4=val4"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3=", "key4=val4"},
	}

	for i := range taskEnvsTests {
		mergeEnvs, _ := mergeEnvs(stepEnvsTests[i], taskEnvsTests[i])
		for j := range mergeEnvs {
			if expects[i][j] != mergeEnvs[j] {
				t.Errorf("running test %v, expected merge of step and task envs to be %v but got %v", i, expects[i], mergeEnvs)
			}
		}
	}
}

func TestNewTaskFromString(t *testing.T) {
	tests := []struct {
		template    string
		secrets     []*Secret
		shouldError bool
	}{
		{`
secrets:
  - id: mysecret
    akv: https://myvault.vault.azure.net/secrets/mysecret
  - id: mysecret1
    akv: https://myvault.vault.azure.net/secrets/mysecret1
    clientID: c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86`,
			[]*Secret{
				{
					ID:  "mysecret",
					Akv: "https://myvault.vault.azure.net/secrets/mysecret",
				},
				{
					ID:          "mysecret1",
					Akv:         "https://myvault.vault.azure.net/secrets/mysecret1",
					MsiClientID: "c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86",
				},
			},
			false,
		},
		{`
secrets:
  - id: MYSecret1
  - id: mysecret1
    clientID: c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86`,
			[]*Secret{},
			true,
		},
		{`
secrets:`,
			[]*Secret{},
			false,
		},
		{``,
			[]*Secret{},
			false,
		},
		{`
secrets:
  - id: mysecret1
    akv: myakv
  - id: mysecret1
    akv: myakv2
    clientID: c72b2df0-b9d8-4ac6-9363-7c1eb06c1c86`,
			[]*Secret{},
			true,
		},
		{`
steps:
  - id: mystep
    cmd: bash echo hello world`,
			[]*Secret{},
			false,
		},
		{`
steps:
  - cmd: bash echo hello world`,
			[]*Secret{},
			false,
		},
	}

	for _, test := range tests {
		task, err := NewTaskFromString(test.template)
		if test.shouldError && err == nil {
			t.Fatalf("Expected task: %v to error but it didn't", test.template)
		}
		if !test.shouldError && err != nil {
			t.Fatalf("Task: %v shouldn't have errored, but it did; err: %v", test.template, err)
		}

		if err == nil {
			if len(task.Secrets) != len(test.secrets) {
				t.Errorf("Expected number of secrets: %v, but got %v", len(test.secrets), len(task.Secrets))
			}
			for i := 0; i < len(task.Secrets); i++ {
				if !task.Secrets[i].Equals(test.secrets[i]) {
					t.Errorf("Expected secrets %v and %v be equal", test.secrets[i], task.Secrets[i])
				}
			}
		}
	}
}

func TestGetValidVersion(t *testing.T) {
	tests := []struct {
		version            string
		expectedNewVersion string
	}{
		// Valid
		{"1.0-preview-1", "1.0-preview-1"},

		// Invalid
		{"", currentTaskVersion},
		{"v1.0.0-alpha", currentTaskVersion},
	}

	for _, test := range tests {
		if actual := getValidVersion(test.version); actual != test.expectedNewVersion {
			t.Errorf("expected %s but got %s", test.expectedNewVersion, actual)
		}
	}
}
