package domain

import (
	"bytes"
	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/constants"
)

type Runner interface {
	AppendContext(newEnv []EnvVar) (Runner, error)
	Resolve(value AbstractString) string
	ExecuteCmd(cmdExe AbstractString, cmdArgs []AbstractString) error
	ExecuteString(cmdString AbstractString) error
	DoesDirExist(path AbstractString) (bool, error)
	DoesFileExist(path AbstractString) (bool, error)
	IsDirEmpty(path AbstractString) (bool, error)
	Chdir(path AbstractString) error
	GetEnv(key string) (string, bool)
}

// TODO: verify all referenced task are defined and of type ReferenceTask or ShellTask
// TODO: verify all registries has a login
type BuildRequest struct {
	Version     AbstractString
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

type EnvExporter interface {
	Export() []EnvVar
}

type Task interface {
	Execute(runner Runner) error
	Append(env []EnvVar, parameters []AbstractString) Task
}

type ReferencedTask struct {
	context    *map[string]Task
	Name       string
	Parameters []AbstractString
	Env        []EnvVar
}

func (t *ReferencedTask) Execute(runner Runner) error {
	reference, found := (*t.context)[t.Name]
	if !found {
		return fmt.Errorf("Undefined task: %s", t.Name)
	}
	return reference.Append(t.Env, t.Parameters).Execute(runner)
}

func (t *ReferencedTask) Append(env []EnvVar, parameters []AbstractString) Task {
	return &ReferencedTask{
		context:    t.context,
		Name:       t.Name,
		Env:        append(t.Env, env...),
		Parameters: append(t.Parameters, parameters...),
	}
}

type ShellTask struct {
	Command AbstractString
	Env     []EnvVar
}

func (t *ShellTask) Execute(runner Runner) error {
	taskRunner, err := runner.AppendContext(t.Env)
	if err != nil {
		return err
	}
	return taskRunner.ExecuteString(t.Command)
}

func (t *ShellTask) Append(env []EnvVar, parameters []AbstractString) Task {
	buf := new(bytes.Buffer)
	buf.WriteString(t.Command.raw)
	for _, s := range parameters {
		buf.WriteByte(' ')
		buf.WriteString(s.raw)
	}
	return &ShellTask{
		Command: *Abstract((string)(buf.Bytes())),
		Env:     append(t.Env, env...),
	}
}

type SourceDescription interface {
	EnsureSource(runner Runner) error
	EnsureBranch(runner Runner, branch AbstractString) error
	Export() []EnvVar
}

type LocalSource struct {
	Dir AbstractString
}

func NewLocalSource(dir string) (*LocalSource, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("Error trying to locate current working directory as local source %s", err)
		}
	}
	return &LocalSource{Dir: *Abstract(dir)}, nil
}

func (s *LocalSource) EnsureSource(runner Runner) error {
	return runner.Chdir(s.Dir)
}

func (s *LocalSource) EnsureBranch(runner Runner, branch AbstractString) error {
	if !branch.IsEmpty() {
		return fmt.Errorf("Static local source does not support branching")
	}
	return nil
}

func (s *LocalSource) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			Name:  constants.CheckoutDirVar,
			Value: s.Dir,
		},
	}
}

type BuildTarget struct {
	Build BuildTask
	Push  PushTask
}

type BuildTask interface {
	// Build task can't be a generic tasks now because it needs to return ImageDependencies
	// If we use docker events to figure out dependencies, we can make build tasks a generic task
	Execute(runner Runner) ([]ImageDependencies, error)
}

type PushTask interface {
	Execute(runner Runner) error
}

func (t *BuildTarget) Export() []EnvVar {
	exports := []EnvVar{}
	appendExports(exports, t.Build)
	appendExports(exports, t.Push)
	return exports
}

type ImageDependencies struct {
	Image             string   `json:"image"`
	BuildDependencies []string `json:"build-dependencies"`
	RuntimeDependency string   `json:"runtime-dependency"`
}

type DockerAuthentication struct {
	Registry AbstractString
	Auth     DockerAuthenticationMethod
}

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
