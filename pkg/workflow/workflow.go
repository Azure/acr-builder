package workflow

import (
	"github.com/Azure/acr-builder/pkg/domain"
)

// EvaluationTask is a task that can be registered to the workflow
// so when the workflow starts it will be injected with the runner
// and output context for editing
type EvaluationTask func(runner domain.Runner, outputContext *OutputContext) error

// RunningTask is similar to ExecutionTask and you can register to the workflow
// except that running task does not care about the output context
type RunningTask func(runner domain.Runner) error

// OutputContext are the output context for a workflow
// currently the only thing it outputs are image dependencies
type OutputContext struct {
	// If golang has generic, this dependency to domain wouldn't even be here
	ImageDependencies []domain.ImageDependencies
}

// executionItem is a unit of work in workflow
type executionItem struct {
	context *domain.BuilderContext
	task    EvaluationTask
}

// Workflow describe units of works that the builder performs
type Workflow struct {
	output OutputContext
	items  []executionItem
}

// NewWorkflow creates an empty workflow with empty context
func NewWorkflow() *Workflow {
	return &Workflow{}
}

// GetOutputs return the build outputs
func (w *Workflow) GetOutputs() *OutputContext {
	return &w.output
}

// Run all items that are previously compiled in the workflow
func (w *Workflow) Run(runner domain.Runner) error {
	for _, item := range w.items {
		runner.SetContext(item.context)
		err := item.task(runner, &w.output)
		if err != nil {
			return err
		}
	}
	return nil
}

// ScheduleEvaluation schedule a execution and its context to run when the workflow runs
func (w *Workflow) ScheduleEvaluation(context *domain.BuilderContext, task EvaluationTask) {
	w.items = append(w.items, executionItem{
		context: context,
		task:    task,
	})
}

// ScheduleRun schedule a RunningTask and its context to run when the workflow runs
func (w *Workflow) ScheduleRun(context *domain.BuilderContext, task RunningTask) {
	w.items = append(w.items, executionItem{
		context: context,
		task: func(runner domain.Runner, outputContext *OutputContext) error {
			return task(runner)
		},
	})
}
