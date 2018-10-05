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
		task := &Task{
			RegistryName:     test.registry,
			RegistryUsername: test.user,
			RegistryPassword: test.pw,
		}
		actual := task.UsingRegistryCreds()
		if test.expected != actual {
			t.Errorf("expected use of registry creds to be %v but got %v", test.expected, actual)
		}
	}
}

func TestMergingEnvs(t *testing.T) {
	stepEnvsTests := [][]string{
		{},
		{"key1=newVal1", "key2=newVal2"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3"},
		{},
		{"key1=newVal1", "key2=newVal2"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3"},
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
		{"key1=newVal1", "key2=newVal2", "key3=newVal3"},
		{"key1=val1", "key2=val2", "key3=val3"},
		{"key1=newVal1", "key2=newVal2", "key3=val3", "key4=val4"},
		{"key1=newVal1", "key2=newVal2", "key3=newVal3", "key4=val4"},
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
