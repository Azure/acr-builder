// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package graph

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

const (
	rootNodeID = "acb_root"
)

// Node represents a vertex in a Dag.
type Node struct {
	Name     string
	Value    *Step
	children map[string]*Node
	mu       sync.Mutex
	degree   int
}

// NewNode creates a new Node based on the provided name and value.
func NewNode(value *Step) *Node {
	return &Node{
		Name:     value.ID,
		Value:    value,
		children: make(map[string]*Node),
		mu:       sync.Mutex{},
		degree:   0,
	}
}

// GetDegree returns the degree of the node.
func (n *Node) GetDegree() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return n.degree
}

// Dag represents a thread safe directed acyclic graph.
type Dag struct {
	Root  *Node
	Nodes map[string]*Node
	mu    sync.Mutex
}

// NewDag creates a new Dag with a root vertex.
func NewDag() *Dag {
	dag := new(Dag)
	dag.Nodes = make(map[string]*Node)
	dag.Root = NewNode(&Step{ID: rootNodeID})
	return dag
}

// NewDagFromTask creates a new Dag based on the specified Task.
func NewDagFromTask(t *Task) (*Dag, error) {
	dag := NewDag()

	var prevStep *Step
	for _, step := range t.Steps {
		if err := step.Validate(); err != nil {
			return dag, err
		}
		if _, err := dag.AddVertex(step); err != nil {
			return dag, err
		}

		// If the step is parallel, add it to the root
		if step.ShouldExecuteImmediately() {
			if err := dag.AddEdge(rootNodeID, step.ID); err != nil {
				return dag, err
			}
		} else if step.HasNoWhen() {
			// If the step has no when, add it to the root or the previous step
			if prevStep == nil {
				if err := dag.AddEdge(rootNodeID, step.ID); err != nil {
					return dag, err
				}
			} else {
				if err := dag.AddEdge(prevStep.ID, step.ID); err != nil {
					return dag, err
				}
			}
		} else {
			// Otherwise, add edges according to when
			for _, dep := range step.When {
				if err := dag.AddEdge(dep, step.ID); err != nil {
					return dag, err
				}
			}
		}

		prevStep = step
	}

	return dag, nil
}

// AddVertex adds a vertex to the Dag with the specified name and value.
func (d *Dag) AddVertex(value *Step) (*Node, error) {
	if value.ID == rootNodeID {
		return nil, fmt.Errorf("%v is a reserved ID, it can't be used", rootNodeID)
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.Nodes[value.ID]; ok {
		return nil, fmt.Errorf("%s already exists as a vertex", value.ID)
	}
	n := NewNode(value)
	d.Nodes[value.ID] = n
	return n, nil
}

// AddEdge adds an edge between from and to.
func (d *Dag) AddEdge(from string, to string) error {
	fromNode, toNode, err := d.validateFromAndTo(from, to)
	if err != nil {
		return err
	}

	fromNode.mu.Lock()
	fromNode.children[to] = toNode
	fromNode.mu.Unlock()

	toNode.mu.Lock()
	toNode.degree++
	toNode.mu.Unlock()

	return nil
}

// RemoveEdge removes the edge between from and to.
func (d *Dag) RemoveEdge(from string, to string) error {
	fromNode, toNode, err := d.validateFromAndTo(from, to)
	if err != nil {
		return err
	}

	fromNode.mu.Lock()
	delete(fromNode.children, to)
	fromNode.mu.Unlock()

	toNode.mu.Lock()
	toNode.degree--
	toNode.mu.Unlock()

	return nil
}

// Children returns the node's children.
func (n *Node) Children() []*Node {
	childNodes := make([]*Node, 0, len(n.children))

	n.mu.Lock()
	defer n.mu.Unlock()
	for _, v := range n.children {
		childNodes = append(childNodes, v)
	}

	return childNodes
}

func (d *Dag) validateFromAndTo(from string, to string) (fromNode *Node, toNode *Node, err error) {
	if from == "" {
		return nil, nil, errors.New("from cannot be empty")
	}
	if to == "" {
		return nil, nil, errors.New("to cannot be empty")
	}
	if from == to {
		return nil, nil, errors.New("from and to cannot be the same")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	if from == rootNodeID {
		fromNode = d.Root
	} else {
		var ok bool
		if fromNode, ok = d.Nodes[from]; !ok {
			return nil, nil, fmt.Errorf("%v does not exist as a vertex [from: %v, to: %v]", from, from, to)
		}
	}
	toNode, ok := d.Nodes[to]
	if !ok {
		return nil, nil, fmt.Errorf("%v does not exist as a vertex [from: %v, to: %v]", to, from, to)
	}
	return fromNode, toNode, nil
}
