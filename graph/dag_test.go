// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	gocontext "context"
	"fmt"
	"testing"
)

func TestDagCreation_ValidFile(t *testing.T) {
	task, err := UnmarshalTaskFromFile(gocontext.Background(), "testdata/acb.yaml", nil)
	if err != nil {
		t.Fatalf("Failed to create task from file. Err: %v", err)
	}

	expectedVersion := "v1.0.0"
	if task.Version != expectedVersion {
		t.Errorf("Expected %s as the version, but got %s", expectedVersion, task.Version)
	}

	pullerStep := &Step{
		ID:                  "puller",
		Cmd:                 "azure/images/docker pull ubuntu",
		EntryPoint:          "someEntryPoint",
		Envs:                []string{"eric=foo", "foo=bar"},
		ExitedWithout:       []int{0, 255},
		StepStatus:          Skipped,
		Timeout:             defaultStepTimeoutInSeconds,
		WorkingDirectory:    "pullDir",
		Network:             DefaultNetworkName,
		Retries:             5,
		RetryDelayInSeconds: 90,
		Pull:                true,
	}

	cStep := &Step{
		ID:                  "C",
		Cmd:                 "blah",
		When:                []string{ImmediateExecutionToken},
		ExitedWith:          []int{0, 1, 2, 3, 4},
		StepStatus:          Skipped,
		Ports:               []string{"8000:8000", "8080:8080"},
		Expose:              []string{"80", "81"},
		Timeout:             defaultStepTimeoutInSeconds,
		Keep:                true,
		Isolation:           "default",
		Network:             DefaultNetworkName,
		Envs:                []string{"foo=taskEnv"},
		RetryDelayInSeconds: defaultStepRetryDelayInSeconds,
	}

	bStep := &Step{
		ID:                              "B",
		When:                            []string{"C"},
		Cmd:                             "azure/images/git clone https://github.com/ehotinger/clone",
		StepStatus:                      Skipped,
		Timeout:                         defaultStepTimeoutInSeconds,
		IgnoreErrors:                    true,
		Network:                         DefaultNetworkName,
		DisableWorkingDirectoryOverride: true,
		Envs:                            []string{"foo=taskEnv"},
		RetryDelayInSeconds:             defaultStepRetryDelayInSeconds,
	}

	fooStep := &Step{
		ID:                  "build-foo",
		Cmd:                 "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/foo --cache-from=ubuntu",
		Envs:                []string{"eric=foo", "foo=taskEnv"},
		When:                []string{"build-qux"},
		StepStatus:          Skipped,
		Timeout:             defaultStepTimeoutInSeconds,
		Network:             DefaultNetworkName,
		RetryDelayInSeconds: defaultStepRetryDelayInSeconds,
	}

	barStep := &Step{
		ID:                  "build-bar",
		Cmd:                 "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/bar --cache-from=ubuntu",
		When:                []string{ImmediateExecutionToken},
		StepStatus:          Skipped,
		Timeout:             defaultStepTimeoutInSeconds,
		Network:             DefaultNetworkName,
		Envs:                []string{"foo=taskEnv"},
		RetryDelayInSeconds: defaultStepRetryDelayInSeconds,
	}

	quxStep := &Step{
		ID:                  "build-qux",
		Cmd:                 "azure/images/acr-builder build -f Dockerfile https://github.com/ehotinger/qux --cache-from=ubuntu",
		When:                []string{ImmediateExecutionToken},
		StepStatus:          Skipped,
		Timeout:             defaultStepTimeoutInSeconds,
		Detach:              true,
		StartDelay:          50,
		Network:             DefaultNetworkName,
		Envs:                []string{"foo=taskEnv"},
		RetryDelayInSeconds: defaultStepRetryDelayInSeconds,
	}

	qazStep := &Step{
		ID:                  "build-qaz",
		Build:               "-f Dockerfile https://github.com/ehotinger/qaz --cache-from=ubuntu",
		StepStatus:          Skipped,
		Timeout:             defaultStepTimeoutInSeconds,
		Privileged:          true,
		User:                "root",
		Network:             "host",
		Envs:                []string{"foo=taskEnv"},
		RetryDelayInSeconds: defaultStepRetryDelayInSeconds,
		Repeat:              2,
	}

	dict := make(map[string]*Step)
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

	err = verifyChildren(rootStepChildren, task.Dag.Root.Children())
	if err != nil {
		t.Fatalf("root node: %v failed child validation. Err: %v", task.Dag.Root.Name, err)
	}

	for k, node := range task.Dag.Nodes {
		if val, ok := dict[k]; ok {
			if !val.Equals(node.Value) {
				t.Fatalf("Step generated from DAG is different than expected step for %v", k)
			}
			switch node.Name {
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
		return fmt.Errorf("expected %v children, actual: %v", lExpected, lActual)
	}

	for _, node := range actual {
		if lookup, ok := expected[node.Name]; ok {
			if !lookup.Equals(node.Value) {
				return fmt.Errorf("node provided: %v does not match the expected: %v", lookup, node.Value)
			}
		} else {
			return fmt.Errorf("node %v was not expected", node.Name)
		}
	}

	return nil
}
