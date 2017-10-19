package shell

import (
	"os/exec"
	"strings"

	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/sirupsen/logrus"
)

type shellRunner struct {
	context *domain.BuilderContext
	fs      *domain.BuildContextAwareFileSystem
}

// NewRunner creates a runner for a given shell with empty context
func NewRunner() domain.Runner {
	context := domain.NewContext([]domain.EnvVar{}, []domain.EnvVar{})
	return &shellRunner{
		context: context,
		fs:      domain.NewBuildContextAwareFileSystem(context),
	}
}

// GetFileSystem returns the file system the that runner is running under
func (r *shellRunner) GetFileSystem() domain.FileSystem {
	return r.fs
}

// GetContext return the current running context
func (r *shellRunner) GetContext() *domain.BuilderContext {
	return r.context
}

// SetContext updates the current running context
func (r *shellRunner) SetContext(context *domain.BuilderContext) {
	r.context = context
	r.fs.SetContext(context)
}

// ExecuteCmdWithCustomLogging runs the given command but use custom logging logic
// this method can be used to hide secrets passed in
func (r *shellRunner) ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	return r.executeCmd(obfuscate, cmdExe, cmdArgs)
}

// ExecuteCmd runs the given command with default logging
func (r *shellRunner) ExecuteCmd(cmdExe string, cmdArgs []string) error {
	return r.executeCmd(nil, cmdExe, cmdArgs)
}

func (r *shellRunner) executeCmd(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	resolvedArgs := make([]string, len(cmdArgs))
	for i, arg := range cmdArgs {
		resolvedArgs[i] = r.context.Expand(arg)
	}
	cmd := exec.Command(r.context.Expand(cmdExe), resolvedArgs...)
	displayValues := resolvedArgs
	if obfuscate != nil {
		displayValues = make([]string, len(resolvedArgs))
		copy(displayValues, resolvedArgs)
		obfuscate(displayValues)
	}
	logrus.Infof("Running command %s %s", cmdExe, strings.Join(displayValues, " "))
	return r.execute(cmd)
}

func (r *shellRunner) execute(cmd *exec.Cmd) error {
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, r.context.Export()...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start command: %s", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Failed to run command: %s", err)
	}
	return nil
}
