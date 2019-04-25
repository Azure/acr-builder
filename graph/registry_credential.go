// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
)

var (
	errInvalidRegName       = errors.New("registry name can't be empty")
	errInvalidUsername      = errors.New("username can't be empty")
	errInvalidPassword      = errors.New("password can't be empty")
	errInvalidIdentity      = errors.New("identity can't be empty")
	errInvalidAadResourceID = errors.New("aadResourceId can't be empty")
	errCouldNotClassify     = errors.New("unable to classify credential into opaque, vault or msi")
)

const (
	// Opaque means username/password are in plain-text
	Opaque = "opaque"
	// VaultSecret means username/password are Azure KeyVault IDs
	VaultSecret = "vaultsecret"
)

// RegistryCredential defines a combination of registry, username and password.
type RegistryCredential struct {
	Registry      string `json:"registry"`
	Username      string `json:"username,omitempty"`
	UsernameType  string `json:"userNameProviderType,omitempty"`
	Password      string `json:"password,omitempty"`
	PasswordType  string `json:"passwordProviderType,omitempty"`
	Identity      string `json:"identity,omitempty"`
	AadResourceID string `json:"aadResourceId,omitempty"`
}

// CreateRegistryCredentialFromString creates a RegistryCredential object from a serialized string.
func CreateRegistryCredentialFromString(str string) (*RegistryCredential, error) {
	var cred RegistryCredential
	if err := json.Unmarshal([]byte(str), &cred); err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal Credentials from string")
	}

	usernameType := strings.ToLower(cred.UsernameType)
	passwordType := strings.ToLower(cred.PasswordType)

	if cred.Registry == "" {
		return nil, errInvalidRegName
	}

	var retVal *RegistryCredential

	isOpaque := usernameType == Opaque && passwordType == Opaque
	hasVaultSecret := usernameType == VaultSecret || passwordType == VaultSecret
	isMSI := usernameType == "" && passwordType == ""

	if isOpaque {
		if cred.Username == "" {
			return nil, errInvalidUsername
		}
		if cred.Password == "" {
			return nil, errInvalidPassword
		}
		retVal = &RegistryCredential{
			Registry:     cred.Registry,
			Username:     cred.Username,
			UsernameType: usernameType,
			Password:     cred.Password,
			PasswordType: passwordType,
		}
	} else if hasVaultSecret {
		if cred.Username == "" {
			return nil, errInvalidUsername
		}
		if cred.Password == "" {
			return nil, errInvalidPassword
		}
		if cred.Identity == "" {
			return nil, errInvalidIdentity
		}
		retVal = &RegistryCredential{
			Registry:     cred.Registry,
			Username:     cred.Username,
			UsernameType: usernameType,
			Password:     cred.Password,
			PasswordType: passwordType,
			Identity:     cred.Identity,
		}
	} else if isMSI {
		if cred.Identity == "" {
			return nil, errInvalidIdentity
		}
		if cred.AadResourceID == "" {
			return nil, errInvalidAadResourceID
		}
		retVal = &RegistryCredential{
			Registry:      cred.Registry,
			Identity:      cred.Identity,
			AadResourceID: cred.AadResourceID,
		}
	} else {
		return nil, errCouldNotClassify
	}

	return retVal, nil
}

// Equals determines whether two RegistryCredentials are equal.
func (s *RegistryCredential) Equals(t *RegistryCredential) bool {
	if s == nil && t == nil {
		return true
	}
	if s == nil || t == nil {
		return false
	}

	return s.Registry == t.Registry &&
		s.Username == t.Username &&
		s.UsernameType == t.UsernameType &&
		s.Password == t.Password &&
		s.PasswordType == t.PasswordType &&
		s.Identity == t.Identity &&
		s.AadResourceID == t.AadResourceID
}
