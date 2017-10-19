package domain

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAppendContext(t *testing.T) {
	err := os.Setenv("TestAppendContext1", "TestAppendContext.Value1")
	assert.Nil(t, err)
	err = os.Setenv("TestAppendContext2", "TestAppendContext.Value2")
	assert.Nil(t, err)
	err = os.Setenv("TestAppendContextDNE", "")
	assert.Nil(t, err)

	userDefined := []EnvVar{
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

	systemGenerated := []EnvVar{
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

	newlyGenerated := []EnvVar{
		{
			Name:  "s3",
			Value: "Value modified",
		},
		{
			Name:  "s4",
			Value: "${u3}",
		},
	}

	// test user defined value inherit from osEnv
	// both positive and negative
	context := NewContext(userDefined, systemGenerated)
	verifyRunnerOriginalValues(t, context)

	newContext := context.Append(newlyGenerated)
	assert.Equal(t, []EnvVar{
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
	}, newContext.userDefined)
	assertSameEnv(t, []EnvVar{
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
	}, newContext.systemGenerated)
	newContextMap := rebuildMapFromExports(t, newContext)
	assert.Equal(t, 7, len(newContextMap))
	assert.Equal(t, "TestAppendContext.Value1", newContextMap["u1"])
	assert.Equal(t, "UserValue2", newContextMap["u2"])
	assert.Equal(t, "", newContextMap["u3"])
	assert.Equal(t, "TestAppendContext.Value1 is set", newContextMap["s1"])
	assert.Equal(t, "TestAppendContext.Value2 is set", newContextMap["s2"])
	assert.Equal(t, "Value modified", newContextMap["s3"])
	assert.Equal(t, "", newContextMap["s4"])

	verifyRunnerOriginalValues(t, context)
}

func rebuildMapFromExports(t *testing.T, context *BuilderContext) map[string]string {
	exports := context.Export()
	result := make(map[string]string, len(exports))
	for _, entry := range exports {
		split := strings.SplitN(entry, "=", 2)
		assert.Equal(t, 2, len(split))
		result[split[0]] = split[1]
	}
	return result
}

func verifyRunnerOriginalValues(t *testing.T, context *BuilderContext) {
	assert.Equal(t, []EnvVar{
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
	}, context.userDefined)
	assertSameEnv(t, []EnvVar{
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
	}, context.systemGenerated)
	assert.Equal(t, 6, len(context.resolvedContext))
	assert.Equal(t, "TestAppendContext.Value1", context.resolvedContext["u1"])
	assert.Equal(t, "UserValue2", context.resolvedContext["u2"])
	assert.Equal(t, "", context.resolvedContext["u3"])
	assert.Equal(t, "TestAppendContext.Value1 is set", context.resolvedContext["s1"])
	assert.Equal(t, "TestAppendContext.Value2 is set", context.resolvedContext["s2"])
	assert.Equal(t, " is not set", context.resolvedContext["s3"])
}

func assertSameEnv(t *testing.T, expected, actual []EnvVar) {
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
