package commands

import (
	"strings"

	"github.com/Azure/acr-builder/pkg/constants"
)

// KeyValueArgumentObfuscator returns a function to scan the argument list and replace the secret argument value with the obfuscation string
func KeyValueArgumentObfuscator(secretArgs []string) func(args []string) {
	return func(args []string) {
		if len(secretArgs) > 0 {
			for i := 0; i < len(args); i++ {
				for j := 0; j < len(secretArgs); j++ {
					if args[i] == secretArgs[j] {
						index := strings.Index(args[i], "=")
						if index >= 0 {
							args[i] = args[i][:index+1] + constants.ObfuscationString
						} else {
							args[i] = constants.ObfuscationString
						}
						break
					}
				}
			}
		}
	}
}
