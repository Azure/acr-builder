// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var (
	errInvalidRegName    = errors.New("registry name can't be empty")
	errInvalidUsername   = errors.New("username can't be empty")
	errInvalidPassword   = errors.New("password can't be empty")
	errInsufficientCreds = errors.New("need to provide registry name, username, and password in a string delimited by ';'")
)

// Credential defines a combination of registry, username and password.
type Credential struct {
	RegistryName     string
	RegistryUsername string
	RegistryPassword string
}

// NewCredential creates a new Credential.
func NewCredential(regName, regUser, regPw string) (*Credential, error) {
	if regName == "" {
		return nil, errInvalidRegName
	}
	if regUser == "" {
		return nil, errInvalidUsername
	}
	if regPw == "" {
		return nil, errInvalidPassword
	}

	return &Credential{
		RegistryName:     regName,
		RegistryUsername: regUser,
		RegistryPassword: regPw,
	}, nil
}

// CreateCredentialFromString creates a Credential object from a specified string.
func CreateCredentialFromString(str string) (*Credential, error) {
	strs := strings.SplitN(str, ";", 3)

	if len(strs) != 3 {
		return nil, errInsufficientCreds
	}

	return NewCredential(strs[0], strs[1], strs[2])
}

func (c Credential) String() string {
	return fmt.Sprintf("%s %s", c.RegistryName, c.RegistryUsername)
}
