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

type Runner struct {
	env   map[string]*domain.AbstractString
	shell *Shell
}

func NewRunner(sh *Shell, env []domain.EnvVar) (domain.Runner, error) {
	runner := &Runner{shell: sh, env: map[string]*domain.AbstractString{}}
	return runner.AppendContext(env)
}

func (r *Runner) AppendContext(newEnv []domain.EnvVar) (domain.Runner, error) {
	allVars := map[string]*domain.AbstractString{}
	for k, v := range r.env {
		allVars[k] = v.Clone()
	}
	for _, v := range newEnv {
		allVars[v.Name] = v.Value.Clone()
	}
	err := reduceEnv(allVars)
	if err != nil {
		return nil, fmt.Errorf("Error evaluating the initial environments: %s", err)
	}
	return &Runner{
		shell: r.shell,
		env:   allVars,
	}, nil
}

func (a *Runner) Resolve(value domain.AbstractString) string {
	value.Resolve(a.env)
	return value.EscapedValue()
}

func (r *Runner) ExecuteCmd(cmdExe domain.AbstractString, cmdArgs []domain.AbstractString) error {
	resolvedArgs := make([]string, len(cmdArgs))
	displayArgs := make([]string, len(cmdArgs))
	if cmdArgs != nil {
		for i, arg := range cmdArgs {
			resolvedArgs[i] = r.Resolve(arg)
			displayArgs[i] = arg.DisplayValue()
		}
	}
	cmd := exec.Command(r.Resolve(cmdExe), resolvedArgs...)
	logrus.Infof("Running command %s %s", cmdExe.DisplayValue(), strings.Join(displayArgs, " "))
	return r.execute(cmd)
}

func (r *Runner) ExecuteString(cmdString domain.AbstractString) error {
	cmd := exec.Command(r.shell.BootstrapExe, r.Resolve(cmdString))
	logrus.Infof("Running predefined command %s", cmdString.DisplayValue())
	return r.execute(cmd)
}

func (r *Runner) GetFileInfo(path domain.AbstractString) (os.FileInfo, error) {
	resolved := r.Resolve(path)
	return os.Stat(resolved)
}

func (r *Runner) Chdir(path domain.AbstractString) error {
	dir := r.Resolve(path)
	logrus.Infof("Chdir to %s", path.DisplayValue())
	err := os.Chdir(dir)
	if err != nil {
		return fmt.Errorf("Error chdir to %s", dir)
	}
	return nil
}

func (r *Runner) DoesDirExist(path domain.AbstractString) (bool, error) {
	return r.lookupPath(path, true)
}

func (r *Runner) DoesFileExist(path domain.AbstractString) (bool, error) {
	return r.lookupPath(path, false)
}

func (r *Runner) IsDirEmpty(path domain.AbstractString) (bool, error) {
	resolved := r.Resolve(path)
	info, err := ioutil.ReadDir(resolved)
	if err != nil {
		return false, err
	}
	return len(info) == 0, nil
}

func (r *Runner) GetEnv(key string) (string, bool) {
	value, found := r.env[key]
	var result string
	if found {
		result = value.EscapedValue()
	} else {
		result = ""
	}
	return result, found
}

func (r *Runner) lookupPath(path domain.AbstractString, isDir bool) (bool, error) {
	fileInfo, err := r.GetFileInfo(path)
	if err == nil {
		if fileInfo.IsDir() == isDir {
			return true, nil
		}
		err = fmt.Errorf("Path is expected to be IsDir: %t", isDir)
	} else if os.IsNotExist(err) {
		err = nil
	} else {
		logrus.Warnf("Unexpected error while getting path: %s", path.DisplayValue())
	}
	return false, err
}

func (r *Runner) execute(cmd *exec.Cmd) error {
	cmd.Env = append(cmd.Env, os.Environ()...)
	for k, v := range r.env {
		// TODO: verify if space works
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v.EscapedValue()))
	}
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("Failed to start command: %s", err)
	}
	if err := cmd.Wait(); err != nil {
		// TODO: check if any stderr causes this to fail
		return fmt.Errorf("Failed to run command: %s", err)
	}
	return nil
}

const MaxReduceLevel = 5

func reduceEnv(env map[string]*domain.AbstractString) error {
	replaced := true
	levelCount := 0
	for replaced {
		if levelCount == MaxReduceLevel {
			return fmt.Errorf("Variable nested for too many levels")
		}
		replaced = false
		for k, v := range env {
			found, err := v.ResolveWithCycleDetection(env, k)
			if err != nil {
				return err
			}
			replaced = replaced || found
		}
		levelCount++
	}
	return nil
}
