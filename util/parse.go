// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package util

import "strings"

var buildArgLookup = map[string]bool{"--build-arg": true}
var tagLookup = map[string]bool{"-t": true, "--tag": true}

// ParseTags parses tags off a command.
func ParseTags(cmd string) []string {
	return parseArgs(cmd, tagLookup)
}

// ParseBuildArgs parses build args off a command.
func ParseBuildArgs(cmd string) []string {
	return parseArgs(cmd, buildArgLookup)
}

// parseArgs parses args off the specified command using the specified lookup.
func parseArgs(cmd string, lookup map[string]bool) []string {
	fields := strings.Fields(cmd)
	prevField := ""
	matches := []string{}
	for _, field := range fields {
		if found := lookup[prevField]; found {
			matches = append(matches, field)
		}
		prevField = field
	}

	return matches
}
