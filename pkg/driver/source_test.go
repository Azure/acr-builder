package driver

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/commands"
	"github.com/Azure/acr-builder/pkg/constants"
)

// happy cases are tested in build_test.go
// we will mainly test negative case

type getSourceTestCase struct {
	workingDir     string
	gitURL         string
	gitBranch      string
	gitHeadRev     string
	gitXToken      string
	gitPATokenUser string
	gitPAToken     string
	webArchive     string
	expectedError  string
	expectedSource build.Source
}

func TestGetSourceWebArchive(t *testing.T) {
	url := "http://localhost/project.tar.gz"
	target := "home"
	expectedSource := commands.NewArchiveSource(url, target)
	testGetSource(t, getSourceTestCase{
		webArchive:     url,
		workingDir:     target,
		expectedSource: expectedSource,
	})
}

func TestAmbiguous(t *testing.T) {
	url := "http://localhost/project.tar.gz"
	testGetSource(t, getSourceTestCase{
		webArchive:    url,
		gitURL:        url,
		expectedError: fmt.Sprintf("^Ambiguous selection on sources, both %s and %s are selected$", constants.SourceNameGit, constants.SourceNameWebArchive),
	})
}

func TestGetSourceLocal(t *testing.T) {
	target := "home"
	testGetSource(t, getSourceTestCase{
		workingDir:     target,
		expectedSource: commands.NewLocalSource(target),
	})
}

func testGetSource(t *testing.T, tc getSourceTestCase) {
	source, err := getSource(tc.workingDir,
		tc.gitURL, tc.gitBranch, tc.gitHeadRev, tc.gitXToken, tc.gitPATokenUser, tc.gitPAToken,
		tc.webArchive)

	if tc.expectedError != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedError), err.Error())
		return
	}

	assert.Nil(t, err)
	assert.Equal(t, tc.expectedSource, source)
}
