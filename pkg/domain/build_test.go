package domain

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

type newEnvTestCase struct {
	name          string
	value         string
	expectedError string
}

func TestNewEnvValid(t *testing.T) {
	testNewEnv(t, newEnvTestCase{
		name:  "someVar",
		value: "val",
	})
}

func TestNewEnvInvalid1(t *testing.T) {
	testNewEnv(t, newEnvTestCase{
		name:          " var",
		value:         "val",
		expectedError: "^Invalid environmental variable name:  var$",
	})
}

func TestNewEnvInvalid2(t *testing.T) {
	testNewEnv(t, newEnvTestCase{
		name:          "0var",
		value:         "val",
		expectedError: "^Invalid environmental variable name: 0var$",
	})
}

func testNewEnv(t *testing.T, tc newEnvTestCase) {
	envVar, err := NewEnvVar(tc.name, tc.value)
	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	} else {
		assert.Nil(t, err)
		assert.Equal(t, tc.name, envVar.Name)
		assert.Equal(t, tc.value, envVar.Value)
	}
}
