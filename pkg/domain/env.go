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
	raw         string
	isSensitive bool
}

func Abstract(s string) *AbstractString {
	return &AbstractString{raw: s}
}

func AbstractSensitive(s string) *AbstractString {
	return &AbstractString{raw: s, isSensitive: true}
}

func AbstractBatch(in []string) []AbstractString {
	result := make([]AbstractString, len(in))
	for i, val := range in {
		result[i] = *Abstract(val)
	}
	return result
}

func (s *AbstractString) Clone() *AbstractString {
	return &AbstractString{raw: s.raw, isSensitive: s.isSensitive}
}

func (s *AbstractString) DisplayValue() string {
	if s.isSensitive {
		return "*****"
	}
	return s.EscapedValue()
}

func (s *AbstractString) IsEmpty() bool {
	return s.raw == ""
}

func (abs *AbstractString) EscapedValue() string {
	return strings.Replace(abs.raw, "\"$\"", "$", -1)
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
		toReplace := strings.Contains(abs.raw, replaceStr)
		if toReplace {
			if disallowedKey != nil && k == *disallowedKey {
				return false, fmt.Errorf("Cycle detected for key:%s", *disallowedKey)
			}
			abs.raw = strings.Replace(abs.raw, replaceStr, v.raw, -1)
			replaced = true
		}
	}
	return replaced, nil
}
