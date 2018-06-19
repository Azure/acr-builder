package graph

import (
	"errors"
	"fmt"
	"sync"

	"github.com/BurntSushi/toml"
)

const (
	// RootNodeID is the root node at the start of the graph.
	RootNodeID = "root"
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
	Nodes map[string]*Node
	mu    sync.Mutex
}

// NewDag creates a new Dag with a root vertex.
func NewDag() *Dag {
	dag := new(Dag)
	dag.Nodes = make(map[string]*Node)

	// Add the root vertex
	n := NewNode(&Step{ID: RootNodeID})
	dag.Nodes[RootNodeID] = n
	return dag
}

// NewDagFromPipeline creates a new Dag based on the specified pipeline.
func NewDagFromPipeline(p *Pipeline) (*Dag, error) {
	dag := NewDag()

	var prevStep *Step
	for _, step := range p.Steps {
		if err := step.Validate(); err != nil {
			return dag, err
		}

		if _, err := dag.AddVertex(step); err != nil {
			return dag, err
		}

		// If the step is parallel, add it to the root
		if step.ShouldExecuteImmediately() {
			if err := dag.AddEdgeFromRoot(step.ID); err != nil {
				return dag, err
			}
		} else if step.HasNoWhen() {
			// If the step has no when, add it to the root or the previous step
			if prevStep == nil {
				if err := dag.AddEdgeFromRoot(step.ID); err != nil {
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

// CreateDagFromFile creates a Dag from the specified file.
func CreateDagFromFile(file string) (*Dag, error) {
	p, err := UnmarshalPipelineFromFile(file)
	if err != nil {
		return nil, err
	}

	return NewDagFromPipeline(p)
}

// UnmarshalPipelineFromString unmarshals a pipeline from a raw string.
func UnmarshalPipelineFromString(data string) (*Pipeline, error) {
	p := &Pipeline{}
	if _, err := toml.Decode(data, p); err != nil {
		return p, err
	}

	p.initialize()
	return p, nil
}

// UnmarshalPipelineFromFile unmarshals a pipeline from a toml file.
func UnmarshalPipelineFromFile(file string) (*Pipeline, error) {
	p := &Pipeline{}

	// Early exit if the error is nil
	if _, err := toml.DecodeFile(file, p); err != nil {
		return p, err
	}

	// Initialize the pipeline to normalize some values.
	p.initialize()
	return p, nil
}

// AddVertex adds a vertex to the Dag with the specified name and value.
func (d *Dag) AddVertex(value *Step) (*Node, error) {
	if value.ID == RootNodeID {
		return nil, fmt.Errorf("%v is a reserved ID, it cannot be used", RootNodeID)
	}

	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.Nodes[value.ID]; exists {
		return nil, fmt.Errorf("%v already exists as a vertex", value.ID)
	}

	n := NewNode(value)
	d.Nodes[value.ID] = n
	return n, nil
}

// AddEdgeFromRoot adds an edge from the root to the specified vertex.
func (d *Dag) AddEdgeFromRoot(to string) error {
	return d.AddEdge(RootNodeID, to)
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

func (d *Dag) validateFromAndTo(from string, to string) (*Node, *Node, error) {
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
	fromNode, ok := d.Nodes[from]
	if !ok {
		return nil, nil, fmt.Errorf("%v does not exist as a vertex [from: %v, to: %v]", from, from, to)
	}

	toNode, ok := d.Nodes[to]
	if !ok {
		return nil, nil, fmt.Errorf("%v does not exist as a vertex [from: %v, to: %v]", to, from, to)
	}

	return fromNode, toNode, nil
}
