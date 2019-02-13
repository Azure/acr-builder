package graph

import (
	"testing"
)

func TestUsingRegistryCreds(t *testing.T) {
	tests := []struct {
		registry string
		user     string
		pw       string
		expected bool
	}{
		{"foo.azurecr.io", "user", "pw", true},
		{"foo.azurecr.io", "user", "", false},
		{"foo.azurecr.io", "", "pw", false},
		{"", "user", "pw", false},
		{"", "user", "", false},
		{"", "", "pw", false},
		{"", "", "", false},
	}

	for _, test := range tests {
		cred, err := NewCredential(test.registry, test.user, test.pw)
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
			Credentials:  []*Credential{cred},
		}
		actual := task.UsingRegistryCreds()
		if test.expected != actual {
			t.Errorf("expected use of registry creds to be %v but got %v", test.expected, actual)
		}
	}
}

func TestNewTask(t *testing.T) {
	tests := []struct {
		steps                []*Step
		secrets              []*Secret
		registry             string
		username             string
		password             string
		okCredentials        bool
		totalTimeout         int
		isBuildTask          bool
		expectedTotalTimeout int
	}{
		{nil, nil, "registry", "username", "password", true, 100, true, 600},
		{[]*Step{}, []*Secret{}, "", "", "", false, 720, false, 720},
	}

	for _, test := range tests {
		cred, err := NewCredential(test.registry, test.username, test.password)

		if !test.okCredentials {
			if err == nil {
				t.Fatalf("Expected to error out, but did not: %v", test)
			}
		}

		task, err := NewTask(test.steps, test.secrets, test.registry, []*Credential{cred}, test.totalTimeout, test.isBuildTask)
		if err != nil {
			t.Fatalf("Unexpected err while creating task: %v", err)
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
		if test.username != "" && task.Credentials[0].RegistryUsername != test.username {
			t.Fatalf("Expected %v as the registry username but got %v", test.username, task.Credentials[0].RegistryUsername)
		}
		if test.password != "" && task.Credentials[0].RegistryPassword != test.password {
			t.Fatalf("Expected %v as the registry password but got %v", test.password, task.Credentials[0].RegistryPassword)
		}
		if task.TotalTimeout != test.expectedTotalTimeout {
			t.Fatalf("Expected %v as the timeout but got %v", test.expectedTotalTimeout, task.TotalTimeout)
		}
		if task.IsBuildTask != test.isBuildTask {
			t.Fatalf("Expected %v as build task but got %v", test.isBuildTask, task.IsBuildTask)
		}
	}
}

func TestInitializeTimeouts(t *testing.T) {
	tests := []struct {
		steps                []*Step
		totalTimeout         int
		stepTimeout          int
		expectedTotalTimeout int
		expectedStepTimeout  int
	}{
		{nil, 0, 0, defaultTotalTimeoutInSeconds, defaultStepTimeoutInSeconds},
		{nil, 15000, 20000, 20000, 20000},
	}

	for _, test := range tests {
		task := &Task{
			Steps:        test.steps,
			TotalTimeout: test.totalTimeout,
			StepTimeout:  test.stepTimeout,
		}
		err := task.initialize()
		if err != nil {
			t.Fatalf("Unexpected err during initialization: %v", err)
		}

		if task.StepTimeout != test.expectedStepTimeout {
			t.Fatalf("Expected %v as the step timeout but got %v", test.expectedStepTimeout, task.StepTimeout)
		}
		if task.TotalTimeout != test.expectedTotalTimeout {
			t.Fatalf("Expected %v as the total timeout but got %v", test.expectedTotalTimeout, task.TotalTimeout)
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
  - id: mystep
    cmd: bash echo hello world
  - id: mystep
    cmd: bash echo hello world`,
			[]*Secret{},
			true,
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
