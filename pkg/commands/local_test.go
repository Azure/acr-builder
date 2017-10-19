package commands

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	test_domain "github.com/Azure/acr-builder/tests/mocks/pkg/domain"
	"github.com/Azure/acr-builder/tests/testCommon"
	"github.com/stretchr/testify/assert"
)

type localSourceTestCase struct {
	path           string
	getWdErr       *error
	expectedChdir  test_domain.ChdirExpectations
	expectedErr    string
	expectedRtnErr string
	expectedEnv    []domain.EnvVar
}

func TestLocalSourceEmptyHappy(t *testing.T) {
	testLocalSource(t, localSourceTestCase{})
}

func TestLocalSourceParamHappy(t *testing.T) {
	testLocalSource(t, localSourceTestCase{
		path: "proj",
		expectedChdir: []test_domain.ChdirExpectation{
			{Path: "proj"},
			{Path: "home"},
		},
		getWdErr: &testCommon.NilError,
		expectedEnv: []domain.EnvVar{
			{Name: constants.ExportsCheckoutDir, Value: "proj"},
		}})
}

func TestLocalSourceGetWdErr(t *testing.T) {
	getwdErr := fmt.Errorf("Failed to get wd")
	testLocalSource(t, localSourceTestCase{
		path:     "proj",
		getWdErr: &getwdErr,
		expectedEnv: []domain.EnvVar{
			{Name: constants.ExportsCheckoutDir, Value: "proj"},
		},
		expectedErr: "^Failed to get wd$",
	})
}

func testLocalSource(t *testing.T, tc localSourceTestCase) {
	source := NewLocalSource(tc.path)
	runner := test_domain.NewMockRunner()
	defer runner.AssertExpectations(t)
	fs := runner.GetFileSystem().(*test_domain.MockFileSystem)
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
