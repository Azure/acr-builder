package shell

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/shhsu/testify/assert"
)

func TestAppendContext(t *testing.T) {
	os.Setenv("TestAppendContext1", "TestAppendContext.Value1")
	os.Setenv("TestAppendContext2", "TestAppendContext.Value2")
	os.Setenv("TestAppendContextDNE", "")

	userDefined := []domain.EnvVar{
		{
			Name:  "u1",
			Value: "${TestAppendContext1}",
		},
		{
			Name:  "u2",
			Value: "UserValue2",
		},
		{
			Name:  "u3",
			Value: "${TestAppendContextDNE}",
		},
	}

	systemGenerated := []domain.EnvVar{
		{
			Name:  "s1",
			Value: "${u1} is set",
		},
		{
			Name:  "s2",
			Value: "${TestAppendContext2} is set",
		},
		{
			Name:  "s3",
			Value: "${TestAppendContextDNE} is not set",
		},
	}

	newlyGenerated := []domain.EnvVar{
		{
			Name:  "s3",
			Value: "Value modified",
		},
		{
			Name:  "s4",
			Value: "${u3}",
		},
	}

	dummyShell := &Shell{BootstrapExe: "something"}
	// test user defined value inherit from osEnv
	// both positive and negative
	runner := NewRunner(dummyShell, userDefined, systemGenerated).(*shellRunner)
	verifyRunnerOriginalValues(t, runner)

	newRunner := runner.AppendContext(newlyGenerated).(*shellRunner)
	assert.Equal(t, []domain.EnvVar{
		{
			Name:  "u1",
			Value: "${TestAppendContext1}",
		},
		{
			Name:  "u2",
			Value: "UserValue2",
		},
		{
			Name:  "u3",
			Value: "${TestAppendContextDNE}",
		},
	}, newRunner.userDefined)
	assertSameEnv(t, []domain.EnvVar{
		{
			Name:  "s1",
			Value: "${u1} is set",
		},
		{
			Name:  "s2",
			Value: "${TestAppendContext2} is set",
		},
		{
			Name:  "s3",
			Value: "Value modified",
		},
		{
			Name:  "s4",
			Value: "${u3}",
		},
	}, newRunner.systemGenerated)
	assert.Equal(t, 7, len(newRunner.resolvedContext))
	assert.Equal(t, "TestAppendContext.Value1", newRunner.resolvedContext["u1"])
	assert.Equal(t, "UserValue2", newRunner.resolvedContext["u2"])
	assert.Equal(t, "", newRunner.resolvedContext["u3"])
	assert.Equal(t, "TestAppendContext.Value1 is set", newRunner.resolvedContext["s1"])
	assert.Equal(t, "TestAppendContext.Value2 is set", newRunner.resolvedContext["s2"])
	assert.Equal(t, "Value modified", newRunner.resolvedContext["s3"])
	assert.Equal(t, "", newRunner.resolvedContext["s4"])

	verifyRunnerOriginalValues(t, runner)
}

func assertSameEnv(t *testing.T, expected, actual []domain.EnvVar) {
	assert.Equal(t, len(expected), len(actual))
	env := map[string]string{}
	for _, entry := range expected {
		env[entry.Name] = entry.Value
	}
	for _, entry := range actual {
		value, found := env[entry.Name]
		assert.True(t, found, "key %s not found", entry.Name)
		assert.Equal(t, value, entry.Value, "key %s, expected: %s, actual: %s", entry.Name, value, entry.Value)
	}
}

func verifyRunnerOriginalValues(t *testing.T, runner *shellRunner) {
	assert.Equal(t, []domain.EnvVar{
		{
			Name:  "u1",
			Value: "${TestAppendContext1}",
		},
		{
			Name:  "u2",
			Value: "UserValue2",
		},
		{
			Name:  "u3",
			Value: "${TestAppendContextDNE}",
		},
	}, runner.userDefined)
	assertSameEnv(t, []domain.EnvVar{
		{
			Name:  "s1",
			Value: "${u1} is set",
		},
		{
			Name:  "s2",
			Value: "${TestAppendContext2} is set",
		},
		{
			Name:  "s3",
			Value: "${TestAppendContextDNE} is not set",
		},
	}, runner.systemGenerated)
	assert.Equal(t, 6, len(runner.resolvedContext))
	assert.Equal(t, "TestAppendContext.Value1", runner.resolvedContext["u1"])
	assert.Equal(t, "UserValue2", runner.resolvedContext["u2"])
	assert.Equal(t, "", runner.resolvedContext["u3"])
	assert.Equal(t, "TestAppendContext.Value1 is set", runner.resolvedContext["s1"])
	assert.Equal(t, "TestAppendContext.Value2 is set", runner.resolvedContext["s2"])
	assert.Equal(t, " is not set", runner.resolvedContext["s3"])
}

func TestChdir(t *testing.T) {
	pwd, err := os.Getwd()
	assert.Nil(t, err)
	defer os.Chdir(pwd)
	runner := NewRunner(&Shell{}, []domain.EnvVar{}, []domain.EnvVar{})
	err = runner.Chdir("..")
	assert.Nil(t, err)
	parent := filepath.Dir(pwd)
	newWD, err := os.Getwd()
	assert.Nil(t, err)
	assert.Equal(t, parent, newWD)
}

func TestChdirFail(t *testing.T) {
	runner := NewRunner(&Shell{}, []domain.EnvVar{}, []domain.EnvVar{})
	err := runner.Chdir("???")
	assert.NotNil(t, err)
	assert.Equal(t, "Error chdir to ???", err.Error())
}

func TestIsDirEmpty(t *testing.T) {
	runner := NewRunner(&Shell{}, []domain.EnvVar{}, []domain.EnvVar{})
	resourceRoot := path.Join("..", "..", "tests", "resources")
	emptyDirPath := path.Join(resourceRoot, "empty-dir")
	emptyDirInfo, err := os.Stat(emptyDirPath)
	if err != nil {
		if os.IsNotExist(err) {
			err := os.Mkdir(emptyDirPath, 0x777)
			defer os.Remove(emptyDirPath)
			if err != nil {
				t.Errorf("Failed to create for empty-dir, test cannot continue")
				t.Fail()
			}
		} else {
			t.Errorf("Failed to stat for empty-dir, test cannot continue")
			t.Fail()
		}
	} else if !emptyDirInfo.IsDir() {
		t.Errorf("empty-dir exists but is a file, test will fail now")
		t.Fail()
	}
	testCase := []struct {
		path           string
		isEmpty        bool
		expectedErrMsg string
	}{
		{
			path:           resourceRoot,
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
	}

	for _, tc := range testCase {
		isEmpty, err := runner.IsDirEmpty(tc.path)
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
	runner := NewRunner(&Shell{}, []domain.EnvVar{}, []domain.EnvVar{})
	dir := path.Join("..", "..", "tests", "resources", "docker-compose")
	file := path.Join(dir, "docker-compose.yml")
	dne := path.Join(dir, "not_here")
	testCase := []struct {
		path     string
		expected bool
		isDir    bool
	}{
		{
			path:     dir,
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
		exists, err := runner.DoesDirExist(tc.path)
		lookupPathAssertHelper(t, exists, err, tc.expected, true, tc.isDir)
		exists, err = runner.DoesFileExist(tc.path)
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
