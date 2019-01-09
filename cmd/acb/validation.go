// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"errors"
	"fmt"

	"github.com/Azure/acr-builder/graph"
)

var isolations = map[string]bool{
	"":        true,
	"hyperv":  true,
	"process": true,
	"default": true,
}

func validateIsolation(isolation string) error {
	if ok := isolations[isolation]; !ok {
		return fmt.Errorf("invalid isolation: %s", isolation)
	}
	return nil
}

// TODO Need to remove this but right now, `username` and `password` are always empty.
func validateRegistryCreds(username string, password string) error {
	if (username == "" && password == "") || (username != "" && password != "") {
		return nil
	}
	return errors.New("when specifying username and password, provide both or neither")
}

func validatePush(push bool, credentials []string) error {
	if push {
		if len(credentials) == 0 {
			return errors.New("when specifying push, username, password, and registry are required")
		}
		if _, err := graph.CreateCredentialFromString(credentials[0]); err != nil {
			return errors.New("when specifying push, username, password, and registry are required in proper format")
		}
	}
	return nil
}
