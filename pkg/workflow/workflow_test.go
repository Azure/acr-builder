package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/stretchr/testify/assert"
)

var stockImgDependencies1 = []build.ImageDependencies{
	{
		Image:             "img1",
		RuntimeDependency: "run1",
		BuildDependencies: []string{},
	},
	{
		Image:             "img1.2",
		RuntimeDependency: "run1.2",
		BuildDependencies: []string{"build1.2"},
	},
}

var stockImgDependencies2 = []build.ImageDependencies{
	{
		Image:             "img2",
		RuntimeDependency: "run2",
		BuildDependencies: []string{"build2", "build2.1"},
	},
}

type uniqueContextGenerator struct {
	counter int
}

var contextGen = &uniqueContextGenerator{}

func (g *uniqueContextGenerator) New() *build.BuilderContext {
	systemGenerated := []build.EnvVar{}
	for i := 0; i < g.counter; i++ {
		systemGenerated = append(systemGenerated, build.EnvVar{Name: "k" + strconv.Itoa(i) + strconv.Itoa(g.counter), Value: strconv.Itoa(g.counter)})
	}
	g.counter++
	return build.NewContext([]build.EnvVar{}, systemGenerated)
}

type scheduleTester interface {
	schedule(t *testing.T, w *Workflow, expectedRunner build.Runner)
}

type runTester struct {
	err error
}

func (m *runTester) schedule(t *testing.T, w *Workflow, expectedRunner build.Runner) {
	expectedContext := contextGen.New()
	w.ScheduleRun(expectedContext, func(runner build.Runner) error {
		assert.Equal(t, expectedRunner, runner)
		if m.err != nil {
			return m.err
		}
		return nil
	})
}

type executionTester struct {
	items []build.ImageDependencies
	err   error
}

func (m *executionTester) schedule(t *testing.T, w *Workflow, expectedRunner build.Runner) {
	expectedContext := contextGen.New()
	w.ScheduleEvaluation(expectedContext, func(runner build.Runner, outputContext *OutputContext) error {
		assert.Equal(t, expectedRunner, runner)
		if m.err != nil {
			return m.err
		}
		outputContext.ImageDependencies = append(outputContext.ImageDependencies, m.items...)
		return nil
	})
}

type runTestCase struct {
	items          []scheduleTester
	expectedOutput OutputContext
	expectedError  string
}

func TestEmptyWorkflow(t *testing.T) {
	testRun(t, runTestCase{})
}

func TestRunWorkflow(t *testing.T) {
	testRun(t, runTestCase{
		items: []scheduleTester{
			&runTester{
			// empty tasks does not affect the outputs and does not return errors
			},
			&executionTester{
				items: stockImgDependencies2,
			},
			&executionTester{},
			&executionTester{
				items: stockImgDependencies1,
			},
			&runTester{},
		},
		expectedOutput: OutputContext{
			ImageDependencies: append(stockImgDependencies2, stockImgDependencies1...),
		},
	})
}

func TestRunWorkflowFailed(t *testing.T) {
	testRun(t, runTestCase{
		items: []scheduleTester{
			&runTester{
			// empty tasks does not affect the outputs and does not return errors
			},
			&executionTester{
				items: stockImgDependencies2,
			},
			&runTester{
				err: fmt.Errorf("boom! Run failed"),
			},
			&executionTester{
				items: stockImgDependencies1,
			},
			&runTester{},
		},
		expectedOutput: OutputContext{
			ImageDependencies: append(stockImgDependencies2, stockImgDependencies1...),
		},
		expectedError: "^boom! Run failed$",
	})
}

func TestRunWorkflowFailed2(t *testing.T) {
	testRun(t, runTestCase{
		items: []scheduleTester{
			&runTester{
			// empty tasks does not affect the outputs and does not return errors
			},
			&executionTester{
				items: stockImgDependencies2,
			},
			&runTester{},
			&executionTester{
				items: stockImgDependencies1,
				err:   fmt.Errorf("boom! Execution failed"),
			},
			&runTester{},
		},
		expectedOutput: OutputContext{
			ImageDependencies: append(stockImgDependencies2, stockImgDependencies1...),
		},
		expectedError: "^boom! Execution failed$",
	})
}

func testRun(t *testing.T, tc runTestCase) {
	w := NewWorkflow()
	runner := new(test.MockRunner)
	runner.UseDefaultFileSystem()
	for _, item := range tc.items {
		item.schedule(t, w, runner)
	}
	err := w.Run(runner)
	if tc.expectedError == "" {
		assert.Nil(t, err)
		assert.Equal(t, tc.expectedOutput, *w.GetOutputs())
	} else {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
	}
}
