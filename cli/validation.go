package cli

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
	if username != "" && password != "" {
		return nil
	}
	return errors.New("when specifying username and password, provide both or neither")
}
