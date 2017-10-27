package commands

import (
	"fmt"
	"regexp"
	"testing"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

type localSourceTestCase struct {
	path           string
	getWdErr       *error
	expectedChdir  test.ChdirExpectations
	expectedErr    string
	expectedRtnErr string
	expectedEnv    []build.EnvVar
}

func TestLocalSourceEmptyHappy(t *testing.T) {
	testLocalSource(t, localSourceTestCase{})
}

func TestLocalSourceParamHappy(t *testing.T) {
	testLocalSource(t, localSourceTestCase{
		path: "proj",
		expectedChdir: []test.ChdirExpectation{
			{Path: "proj"},
			{Path: "home"},
		},
		getWdErr: &testCommon.NilError,
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: "proj"},
		}})
}

func TestLocalSourceGetWdErr(t *testing.T) {
	getwdErr := fmt.Errorf("Failed to get wd")
	testLocalSource(t, localSourceTestCase{
		path:     "proj",
		getWdErr: &getwdErr,
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: "proj"},
		},
		expectedErr: "^Failed to get wd$",
	})
}

func testLocalSource(t *testing.T, tc localSourceTestCase) {
	source := NewLocalSource(tc.path)
	runner := test.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test.MockFileSystem)
	fs.PrepareChdir(tc.expectedChdir)
	if tc.getWdErr != nil {
		fs.On("Getwd").Return("home", *tc.getWdErr).Once()
	}
	defer fs.AssertExpectations(t)
	testCommon.AssertSameEnv(t, tc.expectedEnv, source.Export())
	err := source.Obtain(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
		err := source.Return(runner)
		if tc.expectedRtnErr != "" {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedRtnErr), err.Error())
		} else {
			assert.Nil(t, err)
		}
	}
}
