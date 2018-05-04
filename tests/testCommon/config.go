package testCommon

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// TestConfig is the configuration used for unit tests
type TestConfig struct {
	ProjectRoot string
}

// Config is the instance of test config object loaded form default locations
var Config TestConfig

// MultiStageExampleRoot is the multistage example's root
var MultiStageExampleRoot string

func init() {
	projectRoot, err := getProjectRoot()
	if err != nil {
		panic(err.Error())
	}
	defaultLocation := path.Join(projectRoot, "tests", "resources", "test_config.json")
	configPtr, err := loadFrom(defaultLocation)
	if err != nil {
		panic(err.Error())
	}
	configPtr.ProjectRoot = projectRoot
	Config = *configPtr
	MultiStageExampleRoot = filepath.Join(Config.ProjectRoot, "tests", "resources", "hello-multistage")
}

func loadFrom(location string) (*TestConfig, error) {
	bytes, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, err
	}
	var config TestConfig
	err = json.Unmarshal(bytes, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// getProjectRoot scans and find the root of acr-builder project
func getProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("Error getting pwd: %s", err)
	}
	for {
		parent, name := filepath.Split(dir)
		if name == "acr-builder" {
			break
		}
		parent = filepath.Clean(parent)
		if parent == "" {
			panic("no acr-builder directory find on pwd")
		}
		dir = parent
	}
	return dir, nil
}
