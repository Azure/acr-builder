package domain

import (
	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/constants"
)

// EnvVar defines an environmental variable
type EnvVar struct {
	Name  string // [a-zA-z_][a-zA-z_0-9]*
	Value string
}

// Runner is used to run shell commands
type Runner interface {
	AppendContext(newEnv []EnvVar) Runner
	Resolve(value string) string
	ExecuteCmd(cmdExe string, cmdArgs []string) error
	// Note: ExecuteCmdWithObfuscation allow obfuscating sensitive data such as
	// authentication tokens or passwords not to be shown in logs
	// However, passing these sensitive data through command lines are not
	// quite safe anyway because OS would keep command logs
	// We need to think about the security implications
	ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error
	DoesDirExist(path string) (bool, error)
	DoesFileExist(path string) (bool, error)
	IsDirEmpty(path string) (bool, error)
	Chdir(path string) error
}

// BuildRequest defines a acr-builder build
type BuildRequest struct {
	Version     string
	Global      []EnvVar
	SharedTasks map[string]Task
	DockerAuths []DockerAuthentication
	Source      SourceDescription
	Build       []BuildTarget
}

// EnvExporter are tasks that would export environmental variables
type EnvExporter interface {
	Export() []EnvVar
}

// Task is a generic interface that denotes a unit of work  (currently not used)
type Task interface {
	Execute(runner Runner) error
}

// SourceDescription defines where the source code is and how to fetch the code
type SourceDescription interface {
	EnsureSource(runner Runner) error
	EnsureBranch(runner Runner, branch string) error
	Export() []EnvVar
}

type localSource struct {
	Dir string
}

// NewLocalSource creates an object denoting locally mounted source
func NewLocalSource(dir string) (SourceDescription, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("Error trying to locate current working directory as local source %s", err)
		}
	}
	return &localSource{Dir: dir}, nil
}

func (s *localSource) EnsureSource(runner Runner) error {
	return runner.Chdir(s.Dir)
}

func (s *localSource) EnsureBranch(runner Runner, branch string) error {
	if branch != "" {
		return fmt.Errorf("Static local source does not support branching")
	}
	return nil
}

func (s *localSource) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			Name:  constants.CheckoutDirVar,
			Value: s.Dir,
		},
	}
}

// BuildTarget defines how the docker images are build and pushed
type BuildTarget struct {
	Build BuildTask
	Push  PushTask
}

// BuildTask is the build part of BuildTarget
type BuildTask interface {
	// Build task can't be a generic tasks now because it needs to return ImageDependencies
	// If we use docker events to figure out dependencies, we can make build tasks a generic task
	Execute(runner Runner) ([]ImageDependencies, error)
}

// PushTask is the push part of BuildTarget
type PushTask interface {
	Execute(runner Runner) error
}

// Export method exports environment variables defined in the build and push tasks
func (t *BuildTarget) Export() []EnvVar {
	exports := []EnvVar{}
	exports = appendExports(exports, t.Build)
	exports = appendExports(exports, t.Push)
	return exports
}

// ImageDependencies denotes docker image dependencies
type ImageDependencies struct {
	Image             string   `json:"image"`
	BuildDependencies []string `json:"build-dependencies"`
	RuntimeDependency string   `json:"runtime-dependency"`
}

// DockerAuthentication is the association between a registry and its authentication method
type DockerAuthentication struct {
	Registry string
	Auth     DockerAuthenticationMethod
}

// DockerAuthenticationMethod denote how to authenticate to a docker registry
type DockerAuthenticationMethod interface {
	Execute(runner Runner) error
}

func appendExports(input []EnvVar, obj interface{}) []EnvVar {
	exporter, toExport := obj.(EnvExporter)
	if toExport {
		return append(input, exporter.Export()...)
	}
	return input
}
