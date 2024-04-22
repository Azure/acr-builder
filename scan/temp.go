package scan

import (
	"fmt"
	"testing"
)

func TestGit(t *testing.T) {
	fmt.Println("Welcome to My Go Program!")

	context := "https://github.com/NVIDIA/k8s-device-plugin/archive/refs/tags/v0.14.3.tar.gz"
	_, err := Clone(context, ".")
	if err != nil {
		fmt.Println(err, "unable to git clone")
	}

	fmt.Println("Exiting the program. Goodbye!")
}
