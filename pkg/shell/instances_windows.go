// +build windows

package shell

// Instances on windows
var Instances = map[string]*Shell{
	"cmd": &Shell{
		BootstrapExe: "C:\\Windows\\System32\\cmd.exe",
	},
}
