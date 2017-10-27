package driver

import "testing"
import "github.com/stretchr/testify/assert"

import "fmt"

// happy cases are tested in build_test.go
// we will mainly test negative case

func TestNewFactoryMissingRequired(t *testing.T) {
	name := "test1"
	missing := "missing"
	factory, err := newFactory(name, nil, []parameter{
		{name: "foo", value: "bar"},
		{name: missing},
	}, nil)
	assert.Nil(t, factory)
	assert.Equal(t, fmt.Sprintf("Required parameter %s is not given for %s", missing, name), err.Error())
}

func TestNewFactoryOnlyOptional(t *testing.T) {
	name := "test2"
	factory, err := newFactory(name, nil, nil, []parameter{
		{name: "foo", value: "bar"},
	})
	assert.Nil(t, err)
	assert.True(t, factory.isSelected)
}
