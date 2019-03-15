// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

var (
	errInvalidRegName  = errors.New("registry name can't be empty")
	errInvalidUsername = errors.New("username can't be empty")
	errInvalidPassword = errors.New("password can't be empty")
	errInvalidType     = errors.New("type is either empty or invalid")
	errInvalidIdentity = errors.New("identity can't be empty")
)

const (
	// Opaque means username/password are in plain-text
	Opaque = "opaque"
	// Vault means username/password are Azure KeyVault IDs
	Vault = "vault"
	// MSI means the login is done via Managed Identity
	MSI = "msi"
)

// RegistryCredential defines a combination of registry, username and password.
type RegistryCredential struct {
	Type     string `json:"type"`
	Name     string `json:"registry"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Identity string `json:"identity,omitempty"`
}

// CreateRegistryCredentialFromString creates a RegistryCredential object from a serialized string.
func CreateRegistryCredentialFromString(str string) (*RegistryCredential, error) {
	var cred RegistryCredential
	if err := json.Unmarshal([]byte(str), &cred); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("Unable to unmarshal Credentials from string '%s'", str))
	}

	credType := cred.Type
	regName := cred.Name
	regUser := cred.Username
	regPw := cred.Password
	identity := cred.Identity

	if credType == "" {
		return nil, errInvalidType
	}
	if regName == "" {
		return nil, errInvalidRegName
	}

	var retVal *RegistryCredential
	switch strings.ToLower(credType) {
	case Opaque:
		if regUser == "" {
			return nil, errInvalidUsername
		}
		if regPw == "" {
			return nil, errInvalidPassword
		}
		retVal = &RegistryCredential{
			Type:     Opaque,
			Name:     regName,
			Username: regUser,
			Password: regPw,
		}
	case Vault:
		if regUser == "" {
			return nil, errInvalidUsername
		}
		if regPw == "" {
			return nil, errInvalidPassword
		}
		if identity == "" {
			return nil, errInvalidIdentity
		}
		retVal = &RegistryCredential{
			Type:     Vault,
			Name:     regName,
			Username: regUser,
			Password: regPw,
			Identity: identity,
		}
	case MSI:
		if identity == "" {
			return nil, errInvalidIdentity
		}
		retVal = &RegistryCredential{
			Type:     MSI,
			Name:     regName,
			Identity: identity,
		}
	default:
		return nil, errInvalidType
	}

	return retVal, nil
}
