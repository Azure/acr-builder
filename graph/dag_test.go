// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"testing"
)

// TestDagCreation_ValidFile tests that a valid DAG is created from a file.
func TestDagCreation_ValidFile(t *testing.T) {
	pipeline, err := UnmarshalPipelineFromFile("testdata/rally.yaml", "", "", "")
	if err != nil {
		t.Fatalf("Failed to create pipeline from file. Err: %v", err)
	}

	rootStep := &Step{ID: RootNodeID}

	pullerStep := &Step{
		ID:            "puller",
		Run:           "azure/images/docker pull ubuntu",
		EntryPoint:    "someEntryPoint",
		Envs:          []string{"eric=foo", "foo=bar"},
		SecretEnvs:    []string{"someAkvSecretEnv"},
		ExitedWithout: []int{0, 255},
		StepStatus:    Skipped,
		Timeout:       defaultStepTimeoutInSeconds,
	}

	cStep := &Step{
		ID:         "C",
		Run:        "blah",
		When:       []string{ImmediateExecutionToken},
		ExitedWith: []int{0, 1, 2, 3, 4},
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
		Rm:         true,
	}

	bStep := &Step{
		ID:         "B",
		When:       []string{"C"},
		Run:        "azure/images/git clone https://github.com/ehotinger/clone",
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
	}

	fooStep := &Step{
		ID:         "build-foo",
		Run:        "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/foo --cache-from=ubuntu",
		Envs:       []string{"eric=foo"},
		When:       []string{"build-qux"},
		SecretEnvs: []string{"someAkvSecretEnv"},
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
	}

	barStep := &Step{
		ID:         "build-bar",
		Run:        "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/bar --cache-from=ubuntu",
		When:       []string{ImmediateExecutionToken},
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
	}

	quxStep := &Step{
		ID:         "build-qux",
		Run:        "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/qux --cache-from=ubuntu",
		When:       []string{ImmediateExecutionToken},
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
	}

	qazStep := &Step{
		ID:         "build-qaz",
		Run:        "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/qaz --cache-from=ubuntu",
		StepStatus: Skipped,
		Timeout:    defaultStepTimeoutInSeconds,
	}

	dict := make(map[string]*Step)
	dict[RootNodeID] = rootStep
	dict[pullerStep.ID] = pullerStep
	dict[cStep.ID] = cStep
	dict[bStep.ID] = bStep
	dict[fooStep.ID] = fooStep
	dict[barStep.ID] = barStep
	dict[quxStep.ID] = quxStep
	dict[qazStep.ID] = qazStep

	rootStepChildren := make(map[string]*Step)
	rootStepChildren[pullerStep.ID] = pullerStep
	rootStepChildren[cStep.ID] = cStep
	rootStepChildren[quxStep.ID] = quxStep
	rootStepChildren[barStep.ID] = barStep

	cStepChildren := make(map[string]*Step)
	cStepChildren[bStep.ID] = bStep

	quxStepChildren := make(map[string]*Step)
	quxStepChildren[fooStep.ID] = fooStep

	fooStepChildren := make(map[string]*Step)
	fooStepChildren[qazStep.ID] = qazStep

	noChildren := make(map[string]*Step)

	for k, node := range pipeline.Dag.Nodes {
		if val, ok := dict[k]; ok {
			if !val.Equals(node.Value) {
				t.Fatalf("Step generated from DAG is different than expected step for %v", k)
			}
			var err error
			switch node.Name {
			case rootStep.ID:
				err = verifyChildren(rootStepChildren, node.Children())
			case pullerStep.ID:
				err = verifyChildren(noChildren, node.Children())
			case cStep.ID:
				err = verifyChildren(cStepChildren, node.Children())
			case bStep.ID:
				err = verifyChildren(noChildren, node.Children())
			case fooStep.ID:
				err = verifyChildren(fooStepChildren, node.Children())
			case barStep.ID:
				err = verifyChildren(noChildren, node.Children())
			case quxStep.ID:
				err = verifyChildren(quxStepChildren, node.Children())
			case qazStep.ID:
				err = verifyChildren(noChildren, node.Children())
			default:
				t.Fatalf("Unhandled node: %v", k)
			}

			if err != nil {
				t.Fatalf("%v failed child validation. Err: %v", node.Name, err)
			}

		} else {
			t.Fatalf("Unknown node name: %v", k)
		}
	}
}

func verifyChildren(expected map[string]*Step, actual []*Node) error {
	lExpected := len(expected)
	lActual := len(actual)

	if lExpected != lActual {
		return fmt.Errorf("Expected %v children, actual: %v", lExpected, lActual)
	}

	for _, node := range actual {
		if lookup, ok := expected[node.Name]; ok {
			if !lookup.Equals(node.Value) {
				return fmt.Errorf("Node provided does not match the expected node for %v", node.Name)
			}
		} else {
			return fmt.Errorf("Node %v was not expected", node.Name)
		}
	}

	return nil
}
