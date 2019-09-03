package graph

import (
	"runtime"
)

// isAlphanumeric checks whether a particular rune is alphanumeric
func isAlphanumeric(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')
}

func getTestGlobalDefaults() string {

	var fileLoc string
	if runtime.GOOS == "windows" {
		fileLoc = "./global-defaults-windows.yaml"
	} else { // Looking at Linux
		fileLoc = "./global-defaults-linux.yaml"
	}
	return fileLoc
}
