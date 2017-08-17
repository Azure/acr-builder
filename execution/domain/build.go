package domain

import (
	"bytes"
	"fmt"
)

type Runner interface {
	AppendContext(newEnv []EnvVar) (Runner, error)
	Resolve(value AbstractString) string
	ExecuteCmd(cmdExe AbstractString, cmdArgs ...AbstractString) error
	ExecuteString(cmdString AbstractString) error
	DoesDirExist(path AbstractString) (bool, error)
	DoesFileExist(path AbstractString) (bool, error)
	IsDirEmpty(path AbstractString) (bool, error)
	Chdir(path AbstractString) error
}

// TODO, verify all referenced task are defined and of type ReferenceTask or ShellTask
// TODO, verify all registries has a login
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
	Prepublish  Task
	Postpublish Task
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
	buf.WriteString(t.Command.value)
	for _, s := range parameters {
		buf.WriteByte(' ')
		buf.WriteString(s.value)
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

// Currently not support static local source
// type LocalSource struct {
// 	Dir AbstractString
// }

// func (s *LocalSource) EnsureSource(runner Runner) error {
// 	// TODO: document that every path is relative to source
// 	return runner.Chdir(s.Dir)
// }

// func (s *LocalSource) EnsureBranch(runner Runner, branch AbstractString) error {
// 	if branch.value == "" {
// 		return fmt.Errorf("Source does not support branching")
// 	}
// 	return nil
// }

// func (s *LocalSource) Export() []EnvVar {
// 	return []EnvVar{EnvVar{
// 		Name:  constants.BuildSourceVar,
// 		Value: s.Dir,
// 	}}
// }
