package grok

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
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
func ResolveDockerComposeDependencies(env *domain.BuilderContext, projectDirectory string, composeFile string) ([]domain.ImageDependencies, error) {
	result := []domain.ImageDependencies{}
	compose, err := readDockerComposeFile(composeFile)
	if err != nil {
		return result, err
	}

	if projectDirectory == "" {
		projectDirectory = filepath.Dir(composeFile)
	}

	for _, service := range compose.Services.Services {
		contextDir := env.Expand(service.Build.ContextDir)
		imageContext := filepath.Join(projectDirectory, contextDir)
		var dockerfilePath string
		if service.Build.Dockerfile == "" {
			dockerfilePath = filepath.Join(imageContext, constants.DefaultDockerfile)
		} else {
			dockerfilePath = filepath.Join(imageContext, env.Expand(service.Build.Dockerfile))
		}
		runtime, buildtime, err := ResolveDockerfileDependencies(dockerfilePath)
		if err != nil {
			return nil, fmt.Errorf("Failed to list dependencies for dockerfile %s, error, %s", dockerfilePath, err)
		}
		result = append(result, domain.ImageDependencies{
			Image:             env.Expand(service.Image),
			RuntimeDependency: runtime,
			BuildDependencies: buildtime,
		})
	}
	return result, nil
}
