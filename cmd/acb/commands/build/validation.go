// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package build

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
		return fmt.Errorf("invalid isolation: %s", isolation)
	}
	return nil
}

func validatePush(push bool, credentials []string) error {
	if push && len(credentials) == 0 {
		return errors.New("when specifying push, at least one credential is required")
	}
	return nil
}
