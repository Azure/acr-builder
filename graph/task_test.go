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
		var credentials []*Credential

		credentials = append(credentials, &Credential{
			RegistryName:     test.registry,
			RegistryUsername: test.user,
			RegistryPassword: test.pw,
		})
		task := &Task{
			RegistryName: test.registry,
			Credentials:  credentials,
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
		totalTimeout         int
		isBuildTask          bool
		expectedTotalTimeout int
	}{
		{nil, nil, "registry", "username", "password", 100, true, 600},
		{[]*Step{}, []*Secret{}, "", "", "", 720, false, 720},
	}

	for _, test := range tests {
		var credentials []*Credential

		// First add the creds provided by the user
		credentials = append(credentials, &Credential{
			RegistryName:     test.registry,
			RegistryUsername: test.username,
			RegistryPassword: test.password,
		})
		task, err := NewTask(test.steps, test.secrets, test.registry, credentials, test.totalTimeout, test.isBuildTask)
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
		if task.Credentials[0].RegistryUsername != test.username {
			t.Fatalf("Expected %v as the registry username but got %v", test.username, task.Credentials[0].RegistryUsername)
		}
		if task.Credentials[0].RegistryPassword != test.password {
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

	//Expect: stepEnvs should overwrite envs that exist in taskEnvs
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
