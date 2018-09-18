// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package procmanager

import (
	"context"
	"testing"
)

func TestDryRun(t *testing.T) {
	pm := NewProcManager(true)
	if err := pm.Run(context.Background(), nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
}

func TestRun_NilArgs(t *testing.T) {
	pm := NewProcManager(false)
	if err := pm.Run(context.Background(), nil, nil, nil, nil, ""); err != nil {
		t.Fatalf("Unexpected err: %v", err)
	}
}
