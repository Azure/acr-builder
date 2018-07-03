// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

// SelfReferencedStepError defines an error to be returned if the Step is self-referenced.
type SelfReferencedStepError struct {
	message string
}

// NewSelfReferencedStepError creates a new SelfReferencedStepError with the specified message.
func NewSelfReferencedStepError(message string) *SelfReferencedStepError {
	return &SelfReferencedStepError{
		message: message,
	}
}

// Error returns the error message for a SelfReferencedStepError.
func (e *SelfReferencedStepError) Error() string {
	return e.message
}
