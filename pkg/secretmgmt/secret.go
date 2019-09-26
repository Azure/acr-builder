// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package secretmgmt

import (
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

var (
	errMissingSecretIDs      = errors.New("secret is missing an ID as well as auto-generated ID")
	errMissingSecretProps    = errors.New("secret should contain either keyvault property for vault secret, or msi clientID/aadResourceId for msi authentication")
	errSecretIDContainsSpace = errors.New("secret ID cannot contain spaces")
	errInvalidUUID           = errors.New("msi client ID is not a valid guid")
)

// Secret defines a wrapper to resolve vault secrets to values.
type Secret struct {
	ID          string `yaml:"id"`
	KeyVault    string `yaml:"keyvault,omitempty"`
	MsiClientID string `yaml:"clientID,omitempty"`

	// After the Secret is resolved, the value can be found here.
	ResolvedValue string

	// AadResourceID is used to fetch ARM token from a TokenServer for an identity
	AadResourceID string

	// ResolvedChan is used to signal the callers
	// that the secret has been resolved successfully to a value.
	ResolvedChan chan bool

	// TimeoutChan is used to signal the callers
	// that resolving secret timed out.
	TimeoutChan chan struct{}
}

// Validate validates the secrets and returns an error if the secret properties are invalid.
func (s *Secret) Validate() error {
	if s == nil {
		return nil
	}

	if s.ID == "" {
		return errMissingSecretIDs
	}
	if util.ContainsSpace(s.ID) {
		return errSecretIDContainsSpace
	}
	if !s.IsKeyVaultSecret() && !s.IsMsiSecret() {
		return errMissingSecretProps
	}
	if s.MsiClientID != "" && !util.IsValidUUID(s.MsiClientID) {
		return errInvalidUUID
	}

	return nil
}

// IsKeyVaultSecret returns true if a Secret is a key vault, false otherwise.
func (s *Secret) IsKeyVaultSecret() bool {
	return s != nil && s.KeyVault != ""
}

// IsMsiSecret returns true if a Secret is an MSI, false otherwise.
func (s *Secret) IsMsiSecret() bool {
	return s != nil && s.AadResourceID != ""
}

// Equals determines whether or not two secrets are equal.
func (s *Secret) Equals(t *Secret) bool {
	if s == nil && t == nil {
		return true
	}
	if s == nil || t == nil {
		return false
	}

	return s.ID == t.ID &&
		s.KeyVault == t.KeyVault &&
		s.MsiClientID == t.MsiClientID &&
		s.AadResourceID == t.AadResourceID
}
