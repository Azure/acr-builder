package grok

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	yaml "gopkg.in/yaml.v2"
)

type dockerCompose struct {
	Version  string   `yaml:"version"`
	Services services `yaml:"services"`
}

type services struct {
	Services map[string]service
}

type service struct {
	Build buildDirective `yaml:"build"`
	Image string         `yaml:"image"`
}

type buildDirective struct {
	ContextDir string
	Dockerfile string
}

func (s *services) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	var services map[string]service
	if err := unmarshal(&services); err != nil {
		return err
	}
	s.Services = services
	return nil
}

func (s *buildDirective) UnmarshalYAML(unmarshal func(v interface{}) error) error {
	isDirective, err := s.ParseAsBuildDirective(unmarshal)
	if err != nil {
		if _, ok := err.(*yaml.TypeError); !ok {
			return err
		}
	}
	if !isDirective {
		isContextDir, err := s.ParseAsContextDir(unmarshal)
		if err != nil {
			return err
		}
		if !isContextDir {
			return fmt.Errorf("Unable to parse build directive")
		}
	}
	return nil
}

func (s *buildDirective) ParseAsContextDir(unmarshal func(v interface{}) error) (bool, error) {
	var contextDir string
	if err := unmarshal(&contextDir); err != nil {
		return false, err
	}
	s.ContextDir = contextDir
	return true, nil
}

func (s *buildDirective) ParseAsBuildDirective(unmarshal func(v interface{}) error) (bool, error) {
	var directive struct {
		ContextDir string `yaml:"context"`
		Dockerfile string `yaml:"dockerfile"`
	}
	if err := unmarshal(&directive); err != nil {
		return false, err
	}
	s.ContextDir = directive.ContextDir
	s.Dockerfile = directive.Dockerfile
	return true, nil
}

func readDockerComposeFile(path string) (*dockerCompose, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Error opening docker-compose file %s, error: %s", path, err)
	}
	var compose dockerCompose
	err = yaml.Unmarshal(file, &compose)
	if err != nil {
		return nil, fmt.Errorf("Error reading docker-compose file %s, error: %s", path, err)
	}
	return &compose, nil
}

// ResolveDockerComposeDependencies => given a compose file, resolve its dependencies
func ResolveDockerComposeDependencies(env *build.BuilderContext, projectDirectory string, composeFile string) ([]build.ImageDependencies, error) {
	result := []build.ImageDependencies{}
	compose, err := readDockerComposeFile(composeFile)
	if err != nil {
		return nil, err
	}

	if projectDirectory == "" {
		projectDirectory = filepath.Dir(composeFile)
	}

	for serviceName, service := range compose.Services.Services {
		contextDir := env.Expand(service.Build.ContextDir)
		imageContext := filepath.Join(projectDirectory, contextDir)
		var dockerfilePath string
		if service.Build.Dockerfile == "" {
			dockerfilePath = filepath.Join(imageContext, constants.DefaultDockerfile)
		} else {
			dockerfilePath = filepath.Join(imageContext, env.Expand(service.Build.Dockerfile))
		}
		runtime, buildtimes, err := ResolveDockerfileDependencies(dockerfilePath)
		if err != nil {
			return nil, fmt.Errorf("Failed to list dependencies for dockerfile %s, error, %s", dockerfilePath, err)
		}
		image := service.Image
		if image == "" {
			image = serviceName
		}
		dependencies, err := build.NewImageDependencies(env, image, runtime, buildtimes)
		if err != nil {
			return nil, err
		}
		result = append(result, *dependencies)
	}
	return result, nil
}
