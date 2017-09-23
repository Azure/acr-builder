package domain

import (
	"bytes"
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
	ExecuteString(cmdString string) error
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
	Setup       Task
	Prebuild    Task
	Build       []BuildTarget
	Postbuild   Task
	Validation  Task
	Prepush     Task
	Postpush    Task
	WrapUp      Task
}

// EnvExporter are tasks that would export environmental variables
type EnvExporter interface {
	Export() []EnvVar
}

// Task is a generic interface that denotes a unit of work  (currently not used)
type Task interface {
	Execute(runner Runner) error
	Append(env []EnvVar, parameters []string) Task
}

// ReferencedTask refers to another task by name (currently not used)
type ReferencedTask struct {
	context    *map[string]Task
	Name       string
	Parameters []string
	Env        []EnvVar
}

// Execute executes the task referred by ReferencedTask (currently not used)
func (t *ReferencedTask) Execute(runner Runner) error {
	reference, found := (*t.context)[t.Name]
	if !found {
		return fmt.Errorf("Undefined task: %s", t.Name)
	}
	return reference.Append(t.Env, t.Parameters).Execute(runner)
}

// Append appends environment variable and parameters provided (currently not used)
func (t *ReferencedTask) Append(env []EnvVar, parameters []string) Task {
	return &ReferencedTask{
		context:    t.context,
		Name:       t.Name,
		Env:        append(t.Env, env...),
		Parameters: append(t.Parameters, parameters...),
	}
}

// ShellTask defines a shell command (currently not used)
type ShellTask struct {
	Command string
	Env     []EnvVar
}

// Execute executes the shell command (currently not used)
func (t *ShellTask) Execute(runner Runner) error {
	taskRunner := runner.AppendContext(t.Env)
	return taskRunner.ExecuteString(t.Command)
}

// Append appends environment variable and parameters provided (currently not used)
func (t *ShellTask) Append(env []EnvVar, parameters []string) Task {
	buf := new(bytes.Buffer)
	buf.WriteString(t.Command)
	for _, s := range parameters {
		buf.WriteByte(' ')
		buf.WriteString(s)
	}
	return &ShellTask{
		Command: buf.String(),
		Env:     append(t.Env, env...),
	}
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
	appendExports(exports, t.Build)
	appendExports(exports, t.Push)
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
