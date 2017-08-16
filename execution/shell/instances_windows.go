// +build windows

package shell

var Instances = map[string]*Shell{
	"cmd": &Shell{
		BootstrapExe: "C:\\Windows\\System32\\cmd.exe",
	},
}
