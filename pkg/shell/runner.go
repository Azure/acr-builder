package shell

import (
	"io/ioutil"
	"os/exec"
	"strings"

	"fmt"
	"os"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/sirupsen/logrus"
)

type shellRunner struct {
	userDefined     []domain.EnvVar // user defined variables in its raw form
	systemGenerated []domain.EnvVar // system defined variables
	resolvedContext map[string]string
}

// NewRunner creates a runner for a given shell
func NewRunner(userDefined []domain.EnvVar, systemGenerated []domain.EnvVar) domain.Runner {
	runner := &shellRunner{
		userDefined:     userDefined,
		systemGenerated: systemGenerated,
	}
	return runner.AppendContext(systemGenerated)
}

// AppendContext append environment variables that the commands are run on
func (r *shellRunner) AppendContext(newlyGenerated []domain.EnvVar) domain.Runner {
	resolvedContext := map[string]string{}
	for _, entry := range r.userDefined {
		resolvedContext[entry.Name] = ContextResolve(resolvedContext, entry.Value)
	}
	systemGeneratedMap := map[string]domain.EnvVar{}
	for _, entry := range r.systemGenerated {
		systemGeneratedMap[entry.Name] = entry
	}
	for _, entry := range newlyGenerated {
		systemGeneratedMap[entry.Name] = entry
	}
	systemGenerated := []domain.EnvVar{}
	for _, v := range systemGeneratedMap {
		systemGenerated = append(systemGenerated, v)
		resolvedContext[v.Name] = ContextResolve(resolvedContext, v.Value)
	}
	return &shellRunner{
		userDefined:     r.userDefined,
		systemGenerated: systemGenerated,
		resolvedContext: resolvedContext}
}

// ContextResolve : given a context and a string with reference to env variables, expand it
func ContextResolve(context map[string]string, value string) string {
	return os.Expand(value, func(key string) string {
		if value, ok := context[key]; ok {
			return value
		}
		return os.Getenv(key)
	})
}

// Resolve resolves an string given the runner's environment
func (r *shellRunner) Resolve(value string) string {
	return ContextResolve(r.resolvedContext, value)
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
		resolvedArgs[i] = r.Resolve(arg)
	}
	cmd := exec.Command(r.Resolve(cmdExe), resolvedArgs...)
	displayValues := resolvedArgs
	if obfuscate != nil {
		displayValues = make([]string, len(resolvedArgs))
		copy(displayValues, resolvedArgs)
		obfuscate(displayValues)
	}
	logrus.Infof("Running command %s %s", cmd, strings.Join(displayValues, " "))
	return r.execute(cmd)
}

// Chdir changes current working directory for the runner
func (r *shellRunner) Chdir(path string) error {
	dir := r.Resolve(path)
	logrus.Infof("Chdir to %s", path)
	err := os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("Error chdir to %s", dir)
	}
	return nil
}

// DoesDirExist checks if a given directory exists
func (r *shellRunner) DoesDirExist(path string) (bool, error) {
	return r.lookupPath(path, true)
}

// DoesFileExist checks if a given file exists
func (r *shellRunner) DoesFileExist(path string) (bool, error) {
	return r.lookupPath(path, false)
}

// IsDirEmpty checks if given directory is empty
func (r *shellRunner) IsDirEmpty(path string) (bool, error) {
	resolved := r.Resolve(path)
	info, err := ioutil.ReadDir(resolved)
	if err != nil {
		return false, err
	}
	return len(info) == 0, nil
}

func (r *shellRunner) getFileInfo(path string) (os.FileInfo, error) {
	resolved := r.Resolve(path)
	return os.Stat(resolved)
}

func (r *shellRunner) lookupPath(path string, isDir bool) (bool, error) {
	fileInfo, err := r.getFileInfo(path)
	if err == nil {
		if fileInfo.IsDir() == isDir {
			return true, nil
		}
		err = fmt.Errorf("Path is expected to be IsDir: %t", isDir)
	} else if os.IsNotExist(err) {
		err = nil
	} else {
		logrus.Warnf("Unexpected error while getting path: %s", path)
	}
	return false, err
}

func (r *shellRunner) execute(cmd *exec.Cmd) error {
	cmd.Env = append(cmd.Env, os.Environ()...)
	for k, v := range r.resolvedContext {
		// TODO: think about expanding the raw values to enable nesting scenario
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}
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
