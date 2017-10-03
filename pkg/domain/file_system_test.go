package domain

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sharedContext *BuilderContext
var fs *BuildContextAwareFileSystem

func init() {
	sharedContext = NewContext([]EnvVar{
		{Name: "dne_dir", Value: "???"},
		{Name: "resource_root", Value: filepath.Join("..", "..", "tests", "resources")},
		{Name: "docker_compose_project", Value: filepath.Join("${resource_root}", "docker-compose")},
	}, []EnvVar{})
	fs = &BuildContextAwareFileSystem{}
	fs.SetContext(sharedContext)
}

func TestChdir(t *testing.T) {
	pwd, err := os.Getwd()
	assert.Nil(t, err)
	defer os.Chdir(pwd)
	err = fs.Chdir("..")
	assert.Nil(t, err)
	parent := filepath.Dir(pwd)
	newWD, err := os.Getwd()
	assert.Nil(t, err)
	assert.Equal(t, parent, newWD)
}

func TestChdirFail(t *testing.T) {
	err := fs.Chdir("${dne_dir}")
	assert.NotNil(t, err)
	assert.Equal(t, "Error chdir to ???", err.Error())
}

func TestIsDirEmpty(t *testing.T) {
	emptyDirPath := filepath.Join("${resource_root}", "empty-dir")

	// Test Setup
	emptyDirPathResolved := sharedContext.Expand(emptyDirPath)
	emptyDirInfo, err := os.Stat(emptyDirPathResolved)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(emptyDirPathResolved, 0x777)
			defer os.Remove(emptyDirPathResolved)
			if err != nil {
				t.Errorf("Failed to create %s, test cannot continue", emptyDirPathResolved)
				t.Fail()
			}
		} else {
			t.Errorf("Failed to stat %s, test cannot continue", emptyDirPathResolved)
			t.Fail()
		}
	} else if !emptyDirInfo.IsDir() {
		t.Errorf("empty-dir exists but is a file, test will fail now")
		t.Fail()
	}

	// Actual test
	testCase := []struct {
		path           string
		isEmpty        bool
		expectedErrMsg string
	}{
		{
			path:           "${resource_root}",
			isEmpty:        false,
			expectedErrMsg: "",
		},
		{
			path:           emptyDirPath,
			isEmpty:        true,
			expectedErrMsg: "",
		},
		{
			path:           "dne",
			expectedErrMsg: "^open dne: ",
		},
		{
			path:    ".",
			isEmpty: false,
		},
	}

	for _, tc := range testCase {
		isEmpty, err := fs.IsDirEmpty(tc.path)
		if tc.expectedErrMsg != "" {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedErrMsg), err.Error())
		} else {
			assert.Nil(t, err)
			assert.Equal(t, tc.isEmpty, isEmpty)
		}
	}
}

func TestDoesFileOrDirExist(t *testing.T) {
	file := filepath.Join("${docker_compose_project}", "docker-compose.yml")
	dne := filepath.Join("${docker_compose_project}", "not_here")
	testCase := []struct {
		path     string
		expected bool
		isDir    bool
	}{
		{
			path:     "${docker_compose_project}",
			expected: true,
			isDir:    true,
		},
		{
			path:     file,
			expected: true,
			isDir:    false,
		},
		{
			path:     dne,
			expected: false,
			isDir:    false,
		},
	}

	for _, tc := range testCase {
		exists, err := fs.DoesDirExist(tc.path)
		lookupPathAssertHelper(t, exists, err, tc.expected, true, tc.isDir)
		exists, err = fs.DoesFileExist(tc.path)
		lookupPathAssertHelper(t, exists, err, tc.expected, false, tc.isDir)
	}
}

func lookupPathAssertHelper(t *testing.T, exists bool, err error, expectedToExist, expectedIsDir, actuallyIsDir bool) {
	if expectedIsDir == actuallyIsDir {
		assert.Nil(t, err)
		assert.Equal(t, expectedToExist, exists)
	} else if expectedToExist {
		assert.NotNil(t, err)
		assert.Equal(t, fmt.Sprintf("Path is expected to be IsDir: %t", expectedIsDir), err.Error())
	} else {
		assert.False(t, exists)
		assert.Nil(t, err)
	}
}
