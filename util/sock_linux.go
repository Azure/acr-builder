package util

// GetDockerSock returns a volume mapping to the Docker socket.
func GetDockerSock() string {
	return "/var/run/docker.sock:/var/run/docker.sock"
}
