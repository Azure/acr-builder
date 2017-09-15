package domain

import (
	"fmt"
	"strings"
)

type EnvVar struct {
	Name  string // [a-zA-z_][a-zA-z_0-9]*
	Value AbstractString
}

type AbstractString struct {
	value       string
	isSensitive bool
}

func Abstract(s string) *AbstractString {
	return &AbstractString{value: s}
}

func AbstractSensitive(s string) *AbstractString {
	return &AbstractString{value: s, isSensitive: true}
}

func (abs *AbstractString) DisplayValue() string {
	if abs.isSensitive {
		return "*****"
	}
	return abs.value
}

func (abs *AbstractString) RawValue() string {
	return abs.value
}

func (abs *AbstractString) EscapedValue() string {
	return strings.Replace(abs.value, "\"$\"", "$", -1)
}

func (abs *AbstractString) Resolve(env map[string]*AbstractString) bool {
	result, err := abs.resolve(env, nil)
	if err != nil {
		panic("We should not get an error while resolving abstract string")
	}
	return result
}

func (abs *AbstractString) ResolveWithCycleDetection(env map[string]*AbstractString, disallowedKey string) (bool, error) {
	return abs.resolve(env, &disallowedKey)
}

func (abs *AbstractString) resolve(env map[string]*AbstractString, disallowedKey *string) (bool, error) {
	replaced := false
	for k, v := range env {
		replaceStr := "${" + k + "}"
		toReplace := strings.Contains(abs.value, replaceStr)
		if toReplace {
			if disallowedKey != nil && k == *disallowedKey {
				return false, fmt.Errorf("Cycle detected for key:%s", *disallowedKey)
			}
			abs.value = strings.Replace(abs.value, replaceStr, v.value, -1)
			replaced = true
		}
	}
	return replaced, nil
}
