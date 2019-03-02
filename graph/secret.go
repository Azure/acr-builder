// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"github.com/Azure/acr-builder/util"
	"github.com/pkg/errors"
)

var (
	errMissingSecretID       = errors.New("secret is missing an ID")
	errMissingSecretProps    = errors.New("secret is missing required akv property")
	errSecretIDContainsSpace = errors.New("secret ID cannot contain spaces")
	errInvalidUUID           = errors.New("msi client ID is not a valid guid")
)

// SecretValue represents the resolved value of the Secret.
type SecretValue struct {
	ID    string
	Value string
}

// Secret defines a wrapper to resolve vault secrets to values.
type Secret struct {
	ID          string `yaml:"id"`
	Akv         string `yaml:"akv,omitempty"`
	MsiClientID string `yaml:"clientID,omitempty"`

	// ResolvedChan is used to signal the callers
	// that the secret has been resolved successfully to a value.
	ResolvedChan chan SecretValue

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
		return errMissingSecretID
	}
	if util.ContainsSpace(s.ID) {
		return errSecretIDContainsSpace
	}
	if !s.IsAkvSecret() {
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

// Equals determines whether or not two secrets are equal.
func (s *Secret) Equals(t *Secret) bool {
	if s == nil && t == nil {
		return true
	}
	if s == nil || t == nil {
		return false
	}
	if s.ID != t.ID ||
		s.Akv != t.Akv ||
		s.MsiClientID != t.MsiClientID {
		return false
	}

	return true
}
