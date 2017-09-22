// +build !windows

package shell

// Instances on posix
var Instances = map[string]*Shell{
	"bash": &Shell{
		BootstrapExe: "/bin/bash",
	},
	"sh": &Shell{
		BootstrapExe: "/bin/sh",
	},
}
