package util

import "testing"

func TestGetDockerSock(t *testing.T) {
	expected := "/var/run/docker.sock:/var/run/docker.sock"

	if actual := GetDockerSock(); actual != expected {
		t.Errorf("Expected %s as the docker sock, but got %s", expected, actual)
	}
}
