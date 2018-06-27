// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

// StepStatus is an enum for step statuses.
type StepStatus string

const (
	// InProgress means the step is being processed.
	InProgress StepStatus = "inprogress"

	// Successful means the step completed and was successful.
	Successful StepStatus = "successful"

	// Skipped means the step has been skipped.
	Skipped StepStatus = "skipped"

	// Failed means the step failed because of an error.
	Failed StepStatus = "failed"
)
