package shell

import (
	"bytes"
	"io"
	"os/exec"
	"strings"

	"fmt"
	"os"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/sirupsen/logrus"
)

type shellRunner struct {
	context *build.BuilderContext
	fs      *build.ContextAwareFileSystem
}

// NewRunner creates a runner for a given shell with empty context
func NewRunner() build.Runner {
	context := build.NewContext([]build.EnvVar{}, []build.EnvVar{})
	return &shellRunner{
		context: context,
		fs:      build.NewContextAwareFileSystem(context),
	}
}

// GetFileSystem returns the file system the that runner is running under
func (r *shellRunner) GetFileSystem() build.FileSystem {
	return r.fs
}

// GetContext return the current running context
func (r *shellRunner) GetContext() *build.BuilderContext {
	return r.context
}

// SetContext updates the current running context
func (r *shellRunner) SetContext(context *build.BuilderContext) {
	r.context = context
	r.fs.SetContext(context)
}

// ExecuteCmdWithCustomLogging runs the given command but use custom logging logic
// this method can be used to hide secrets passed in
func (r *shellRunner) ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error {
	_, err := r.executeCmd(obfuscate, cmdExe, cmdArgs, false)
	return err
}

// ExecuteCmd runs the given command with default logging
func (r *shellRunner) ExecuteCmd(cmdExe string, cmdArgs []string) error {
	_, err := r.executeCmd(nil, cmdExe, cmdArgs, false)
	return err
}

// ExecuteCmd runs the given command with default logging
func (r *shellRunner) QueryCmd(cmdExe string, cmdArgs []string) (string, error) {
	return r.executeCmd(nil, cmdExe, cmdArgs, true)
}

func (r *shellRunner) executeCmd(obfuscate func([]string), cmdExe string, cmdArgs []string, readOutputs bool) (outputs string, err error) {
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
	var output io.Writer
	var buffer bytes.Buffer
	if readOutputs {
		output = &buffer
	}
	err = r.execute(cmd, output)
	if err != nil {
		return
	}
	if readOutputs {
		outputs = buffer.String()
	}
	return
}

func (r *shellRunner) execute(cmd *exec.Cmd, output io.Writer) error {
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Env = append(cmd.Env, r.context.Export()...)
	cmd.Stdin = os.Stdin
	if output != nil {
		cmd.Stdout = io.MultiWriter(os.Stdout, output)
	} else {
		cmd.Stdout = os.Stdout
	}
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start command: %s", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("Failed to run command: %s", err)
	}
	return nil
}
