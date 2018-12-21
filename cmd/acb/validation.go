// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package main

import (
	"errors"
	"fmt"
)

var isolations = map[string]bool{
	"":        true,
	"hyperv":  true,
	"process": true,
	"default": true,
}

func validateIsolation(isolation string) error {
	if ok := isolations[isolation]; !ok {
		return fmt.Errorf("Invalid isolation: %s", isolation)
	}
	return nil
}

func validateRegistryCreds(username string, password string) error {
	if (username == "" && password == "") || (username != "" && password != "") {
		return nil
	}
	return errors.New("when specifying username and password, provide both or neither")
}

func validatePush(push bool, registry string, username string, password string) error {
	if push && (username == "" || password == "" || registry == "") {
		return errors.New("when specifying push, username, password, and registry are required")
	}
	return nil
}
