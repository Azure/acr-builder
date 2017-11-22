package build

import (
	"fmt"
	"os"
)

// Runner is used to run shell commands
type Runner interface {
	GetFileSystem() FileSystem
	SetContext(context *BuilderContext)
	GetContext() *BuilderContext
	ExecuteCmd(cmdExe string, cmdArgs []string) error
	// Note: ExecuteCmdWithObfuscation allow obfuscating sensitive data such as
	// authentication tokens or passwords not to be shown in logs
	// However, passing these sensitive data through command lines are not
	// quite safe anyway because OS would keep command logs
	// We need to think about the security implications
	ExecuteCmdWithObfuscation(obfuscate func([]string), cmdExe string, cmdArgs []string) error
}

// BuilderContext is the context for Runners
type BuilderContext struct {
	userDefined     []EnvVar // user defined variables in its raw form
	systemGenerated []EnvVar // system defined variables
	resolvedContext map[string]string
}

// NewContext creates a new running context
func NewContext(userDefined []EnvVar, systemGenerated []EnvVar) *BuilderContext {
	context := &BuilderContext{
		userDefined:     userDefined,
		systemGenerated: systemGenerated,
	}
	return context.Append(systemGenerated)
}

// Append append environment variables that the commands are run on
func (r *BuilderContext) Append(newlyGenerated []EnvVar) *BuilderContext {
	resolvedContext := map[string]string{}
	for _, entry := range r.userDefined {
		resolvedContext[entry.Name] = ExpandFromContext(resolvedContext, entry.Value)
	}
	systemGeneratedMap := map[string]EnvVar{}
	for _, entry := range r.systemGenerated {
		systemGeneratedMap[entry.Name] = entry
	}
	for _, entry := range newlyGenerated {
		systemGeneratedMap[entry.Name] = entry
	}
	systemGenerated := []EnvVar{}
	for _, v := range systemGeneratedMap {
		systemGenerated = append(systemGenerated, v)
		resolvedContext[v.Name] = ExpandFromContext(resolvedContext, v.Value)
	}
	return &BuilderContext{
		userDefined:     r.userDefined,
		systemGenerated: systemGenerated,
		resolvedContext: resolvedContext}
}

// ExpandFromContext : given a context and a string with reference to env variables, expand it
func ExpandFromContext(context map[string]string, value string) string {
	return os.Expand(value, func(key string) string {
		if value, ok := context[key]; ok {
			return value
		}
		return os.Getenv(key)
	})
}

// Expand expands an string given the runner's environment
func (r *BuilderContext) Expand(value string) string {
	return ExpandFromContext(r.resolvedContext, value)
}

// Export the environment variables to child cmd to use
func (r *BuilderContext) Export() []string {
	result := make([]string, 0, len(r.resolvedContext))
	for k, v := range r.resolvedContext {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}
