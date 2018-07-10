package graph

import "testing"

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
		p := &Pipeline{
			RegistryName:     test.registry,
			RegistryUsername: test.user,
			RegistryPassword: test.pw,
		}
		actual := p.UsingRegistryCreds()
		if test.expected != actual {
			t.Errorf("expected use of registry creds to be %v but got %v", test.expected, actual)
		}
	}
}
