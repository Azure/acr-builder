// +build !windows

package shell

var Instances = map[string]*Shell{
	"bash": &Shell{
		BootstrapExe: "/bin/bash",
	},
	"sh": &Shell{
		BootstrapExe: "/bin/sh",
	},
}
