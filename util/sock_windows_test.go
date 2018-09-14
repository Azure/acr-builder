package util

import "testing"

func TestGetDockerSock(t *testing.T) {
	expected := "\\\\.\\pipe\\docker_engine:\\\\.\\pipe\\docker_engine"

	if actual := GetDockerSock(); actual != expected {
		t.Errorf("Expected %s as the docker sock, but got %s", expected, actual)
	}
}
