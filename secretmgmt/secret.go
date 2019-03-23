// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package secretmgmt

import (
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

var (
	errMissingSecretIDs      = errors.New("secret is missing an ID as well as auto-generated ID")
	errMissingSecretProps    = errors.New("secret should contain either akv property for vault secret, or msi clientID/armResource for msi authentication")
	errSecretIDContainsSpace = errors.New("secret ID cannot contain spaces")
	errInvalidUUID           = errors.New("msi client ID is not a valid guid")
)

// Secret defines a wrapper to resolve vault secrets to values.
type Secret struct {
	ID          string `yaml:"id"`
	Akv         string `yaml:"akv,omitempty"`
	MsiClientID string `yaml:"clientID,omitempty"`

	// After the Secret is resolved, the value can be found here.
	ResolvedValue string

	// ArmResource is used to fetch ARM token from a TokenServer for an identity
	ArmResource string

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
	if !s.IsAkvSecret() && !s.IsMsiSecret() {
		return errMissingSecretProps
	}
	if s.MsiClientID != "" && !util.IsValidUUID(s.MsiClientID) {
		return errInvalidUUID
	}

	return nil
}

// IsAkvSecret returns true if a Secret is of Azure Keyvault type, false otherwise.
func (s *Secret) IsAkvSecret() bool {
	if s == nil {
		return false
	}
	return s.Akv != ""
}

// IsMsiSecret returns true if a Secret is of MSI type, false otherwise.
func (s *Secret) IsMsiSecret() bool {
	if s == nil {
		return false
	}
	return s.ArmResource != ""
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
		s.Akv == t.Akv &&
		s.MsiClientID == t.MsiClientID &&
		s.ArmResource == t.ArmResource
}
