package commands

import (
	"fmt"
	"regexp"
	"testing"

	test "github.com/Azure/acr-builder/tests/mocks/pkg"
	"github.com/Azure/acr-builder/tests/testCommon"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/stretchr/testify/assert"
)

const gitSourceTestPwd = "get_test_wd"

type gitSourceTestCase struct {
	creds     GitCredential
	address   string
	branch    string
	headRev   string
	targetDir string
	getwdErr  *error
	// TODO: Actually, the expectations should be ordered
	// right now we are not testing if the commands are executed in order
	expectedChdir     test.ChdirExpectations
	expectedFSAccess  test.FileSystemExpectations
	expectedCommands  []test.CommandsExpectation
	expectedEnv       []build.EnvVar
	expectedErr       string
	expectedReturnErr string
}

func TestMinimalParams(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address: "https://github.com/org/address.git",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty(defaultTargetDir, false, nil),
		getwdErr: &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: defaultTargetDir},
			{Path: gitSourceTestPwd},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "git",
				Args:    []string{"clean", "-xdf"},
			},
			{
				Command: "git",
				Args:    []string{"reset", "--hard", "HEAD"},
			},
			{
				Command:      "git",
				Args:         []string{"fetch", "https://github.com/org/address.git"},
				IsObfuscated: true,
			},
			{
				Command:      "git",
				Args:         []string{"pull", "https://github.com/org/address.git"},
				IsObfuscated: true,
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: defaultTargetDir},
			{Name: constants.ExportsGitSource, Value: "https://github.com/org/address.git"},
		},
	})
}

func TestXTokenFreshClone(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		creds:     NewGitXToken("token_value"),
		address:   "https://github.com/org/address.git",
		branch:    "git_branch",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: "target_dir"},
			{Path: gitSourceTestPwd},
		},
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertDirExists("target_dir", false, nil),
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "-b", "git_branch", "https://x-access-token:token_value@github.com/org/address.git", "target_dir"},
				IsObfuscated: true,
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitAuthType, Value: "Git X Token"},
			{Name: constants.ExportsGitSource, Value: "https://github.com/org/address.git"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
		},
	})
}

func TestPATokenRefreshSuccessButFailedToReturn(t *testing.T) {
	creds, err := NewGitPersonalAccessToken("user", "password")
	assert.Nil(t, err)
	testGitSource(t, gitSourceTestCase{
		creds:   creds,
		address: "https://github.com/org/address.git",
		// branch and head rev are both given, prefer headRev
		branch:    "git_branch",
		headRev:   "git_head_rev",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: "target_dir"},
			{Path: gitSourceTestPwd, Err: fmt.Errorf("Error switching back")},
		},
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty("target_dir", false, nil),
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "git",
				Args:    []string{"clean", "-xdf"},
			},
			{
				Command: "git",
				Args:    []string{"reset", "--hard", "HEAD"},
			},
			{
				Command:      "git",
				Args:         []string{"fetch", "https://user:password@github.com/org/address.git"},
				IsObfuscated: true,
			},
			{
				Command: "git",
				Args:    []string{"checkout", "git_head_rev"},
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitAuthType, Value: "Git Personal Access Token"},
			{Name: constants.ExportsGitUser, Value: "user"},
			{Name: constants.ExportsGitSource, Value: "https://github.com/org/address.git"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
			{Name: constants.ExportsGitHeadRev, Value: "git_head_rev"},
		},
		expectedReturnErr: "^Error switching back$",
	})
}

func TestNoAuthHappyClone(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address: "test_address",
		branch:  "git_branch",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty(defaultTargetDir, true, nil),
		getwdErr: &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: defaultTargetDir},
			{Path: gitSourceTestPwd},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "-b", "git_branch", "test_address", defaultTargetDir},
				IsObfuscated: true,
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: defaultTargetDir},
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
		},
	})
}

func TestNoAuthHappyCheckout(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address: "test_address",
		branch:  "git_branch",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty(defaultTargetDir, false, nil),
		getwdErr: &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: defaultTargetDir},
			{Path: gitSourceTestPwd},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "git",
				Args:    []string{"clean", "-xdf"},
			},
			{
				Command: "git",
				Args:    []string{"reset", "--hard", "HEAD"},
			},
			{
				Command:      "git",
				Args:         []string{"fetch", "test_address"},
				IsObfuscated: true,
			},
			{
				Command: "git",
				Args:    []string{"checkout", "git_branch"},
			},
			{
				Command:      "git",
				Args:         []string{"pull", "test_address", "git_branch"},
				IsObfuscated: true,
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: defaultTargetDir},
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
		},
	})
}

func TestNoAuthCloneWithHeadRevFailed(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address: "test_address",
		headRev: "git_head_rev",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty(defaultTargetDir, true, nil),
		getwdErr: &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{Path: defaultTargetDir},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "test_address", defaultTargetDir},
				IsObfuscated: true,
			},
			{
				Command:  "git",
				Args:     []string{"checkout", "git_head_rev"},
				ErrorMsg: "some err",
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsWorkingDir, Value: defaultTargetDir},
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitHeadRev, Value: "git_head_rev"},
		},
		expectedErr: "^Failed checkout git repository at: git_head_rev, error: some err$",
	})
}

func TestNoAuthCloneFailedToFindTargetDir(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		headRev:   "git_head_rev",
		targetDir: "target_dir",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertDirExists("target_dir", false, fmt.Errorf("Some lstat error")),
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitHeadRev, Value: "git_head_rev"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^Error checking for source dir: target_dir, error: Some lstat error$",
	})
}

func TestNoAuthCloneFailedToCheckTargetDirEmpty(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		headRev:   "git_head_rev",
		targetDir: "target_dir",
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty("target_dir", false, fmt.Errorf("some io error")),
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitHeadRev, Value: "git_head_rev"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^Error checking if source dir is empty: target_dir, error: some io error$",
	})
}

func TestNoTokenChdirFailedAfterClone(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		branch:    "git_branch",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedChdir: []test.ChdirExpectation{
			{
				Path: "target_dir",
				Err:  fmt.Errorf("failed to chdir"),
			},
		},
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertDirExists("target_dir", false, nil),
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "-b", "git_branch", "test_address", "target_dir"},
				IsObfuscated: true,
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^failed to chdir$",
	})
}

func TestNoTokenChdirFailedBeforeRefresh(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		branch:    "git_branch",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty("target_dir", false, nil),
		expectedChdir: []test.ChdirExpectation{
			{
				Path: "target_dir",
				Err:  fmt.Errorf("failed to chdir"),
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^failed to chdir$",
	})
}

func TestNoTokenChdirFailedToClean(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		branch:    "git_branch",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty("target_dir", false, nil),
		expectedChdir: []test.ChdirExpectation{
			{
				Path: "target_dir",
			},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command:  "git",
				Args:     []string{"clean", "-xdf"},
				ErrorMsg: "failed to clean",
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^Failed to clean repository: failed to clean$",
	})
}

func TestNoTokenChdirFailedToCheckout(t *testing.T) {
	testGitSource(t, gitSourceTestCase{
		address:   "test_address",
		branch:    "git_branch",
		targetDir: "target_dir",
		getwdErr:  &testCommon.NilError,
		expectedFSAccess: make(test.FileSystemExpectations, 0).
			AssertIsDirEmpty("target_dir", false, nil),
		expectedChdir: []test.ChdirExpectation{
			{
				Path: "target_dir",
			},
		},
		expectedCommands: []test.CommandsExpectation{
			{
				Command: "git",
				Args:    []string{"clean", "-xdf"},
			},
			{
				Command: "git",
				Args:    []string{"reset", "--hard", "HEAD"},
			},
			{
				Command:      "git",
				Args:         []string{"fetch", "test_address"},
				IsObfuscated: true,
				ErrorMsg:     "some network issue",
			},
		},
		expectedEnv: []build.EnvVar{
			{Name: constants.ExportsGitSource, Value: "test_address"},
			{Name: constants.ExportsGitBranch, Value: "git_branch"},
			{Name: constants.ExportsWorkingDir, Value: "target_dir"},
		},
		expectedErr: "^Failed to clean fetch from remote: test_address, error: some network issue$",
	})
}

func testGitSource(t *testing.T, tc gitSourceTestCase) {
	source := NewGitSource(tc.address, tc.branch, tc.headRev, tc.targetDir, tc.creds)
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	fs := runner.GetFileSystem().(*test.MockFileSystem)
	fs.PrepareFileSystem(tc.expectedFSAccess)
	if tc.getwdErr != nil {
		fs.On("Getwd").Return(gitSourceTestPwd, *tc.getwdErr).Once()
	}
	fs.PrepareChdir(tc.expectedChdir)
	defer fs.AssertExpectations(t)
	defer runner.AssertExpectations(t)
	exports := source.Export()
	testCommon.AssertSameEnv(t, tc.expectedEnv, exports)
	err := source.Obtain(runner)
	if tc.expectedErr != "" {
		assert.NotNil(t, err)
		assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
	} else {
		assert.Nil(t, err)
		err := source.Return(runner)
		if tc.expectedReturnErr != "" {
			assert.NotNil(t, err)
			assert.Regexp(t, regexp.MustCompile(tc.expectedReturnErr), err.Error())
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestPATokenNoUsernameOrPassword(t *testing.T) {
	usernames := []string{"user", ""}
	passwords := []string{"pw", ""}
	for _, user := range usernames {
		for _, pw := range passwords {
			token, err := NewGitPersonalAccessToken(user, pw)
			if (user == "") == (pw == "") {
				assert.Nil(t, err)
				creds := token.(*gitPersonalAccessToken)
				assert.Equal(t, user, creds.user)
				assert.Equal(t, pw, creds.token)
			} else {
				assert.NotNil(t, err)
				assert.Regexp(t, regexp.MustCompile(fmt.Sprintf("^Please provide both --%s and --%s or neither$", constants.ArgNameGitPATokenUser, constants.ArgNameGitPAToken)), err.Error())
			}
		}
	}
}

type obfuscationTestCase struct {
	before      []string
	after       []string
	address     string
	obfuscation string
}

func TestGitAddressObfuscatorSingleReplace(t *testing.T) {
	testGitAddressObfuscator(t, obfuscationTestCase{
		before:      []string{"some_addr"},
		after:       []string{"not_there"},
		address:     "some_addr",
		obfuscation: "not_there",
	})
}

func TestGitAddressObfuscatorEmpty(t *testing.T) {
	testGitAddressObfuscator(t, obfuscationTestCase{
		before:      []string{},
		after:       []string{},
		address:     "some_addr",
		obfuscation: "not_there",
	})
}

func TestGitAddressObfuscatorMultipleReplace(t *testing.T) {
	testGitAddressObfuscator(t, obfuscationTestCase{
		before:      []string{"foo", "some_addr", "bar", "some_addr"},
		after:       []string{"foo", "not_there", "bar", "not_there"},
		address:     "some_addr",
		obfuscation: "not_there",
	})
}

func TestGitAddressObfuscatorNoReplace(t *testing.T) {
	testGitAddressObfuscator(t, obfuscationTestCase{
		before:      []string{"foo", "bar"},
		after:       []string{"foo", "bar"},
		address:     "some_addr",
		obfuscation: "not_there",
	})
}

func testGitAddressObfuscator(t *testing.T, tc obfuscationTestCase) {
	obfFunc := gitAddressObfuscator(tc.address, tc.obfuscation)
	obfFunc(tc.before)
	assert.Equal(t, tc.after, tc.before)
}

///////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// The following classes are helper classes that tests some error case not covered by pervious gitSourceTestCase

type simpleGitOperationTestCase struct {
	cred             GitCredential
	address          string
	branch           string
	headRev          string
	targetDir        string
	expectedCommands []test.CommandsExpectation
	expectedErr      string
}

func TestCloneWithInvalidPAAuthAddress(t *testing.T) {
	cred, err := NewGitPersonalAccessToken("user", "pw")
	assert.Nil(t, err)
	testClone(t, simpleGitOperationTestCase{
		cred:        cred,
		branch:      "git_branch",
		address:     "git_address",
		expectedErr: "^Failed to get authorized address, error: Git repository address git_address cannot be used with Access Tokens$",
	})
}

func TestCloneCommandFailed(t *testing.T) {
	testClone(t, simpleGitOperationTestCase{
		branch:  "git_branch",
		address: "git_address",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "-b", "git_branch", "git_address", defaultTargetDir},
				IsObfuscated: true,
				ErrorMsg:     "failed to clone",
			},
		},
		expectedErr: fmt.Sprintf("^Error cloning git source: git_address to directory %s, error: failed to clone$", defaultTargetDir),
	})
}

func TestCloneCommandFailed2(t *testing.T) {
	testClone(t, simpleGitOperationTestCase{
		branch:    "git_branch",
		address:   "git_address",
		targetDir: "target_dir",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"clone", "-b", "git_branch", "git_address", "target_dir"},
				IsObfuscated: true,
				ErrorMsg:     "failed to clone",
			},
		},
		expectedErr: "^Error cloning git source: git_address to directory target_dir, error: failed to clone$",
	})
}

func TestCheckoutAuthAddressFail(t *testing.T) {
	testCheckout(t, simpleGitOperationTestCase{
		cred:        NewGitXToken("token"),
		address:     "invalid_address",
		headRev:     "some_rev",
		expectedErr: "^Failed to get authorized address, error: Git repository address invalid_address cannot be used with Access Tokens$",
	})
}

func TestCheckoutFail(t *testing.T) {
	testCheckout(t, simpleGitOperationTestCase{
		address: "git_address",
		branch:  "git_branch",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"fetch", "git_address"},
				IsObfuscated: true,
			},
			{
				Command:  "git",
				Args:     []string{"checkout", "git_branch"},
				ErrorMsg: "checkout error",
			},
		},
		expectedErr: "^Failed checkout git repository at: git_branch, error: checkout error$",
	})
}

func TestPullFail(t *testing.T) {
	testCheckout(t, simpleGitOperationTestCase{
		address: "git_address",
		branch:  "git_branch",
		expectedCommands: []test.CommandsExpectation{
			{
				Command:      "git",
				Args:         []string{"fetch", "git_address"},
				IsObfuscated: true,
			},
			{
				Command: "git",
				Args:    []string{"checkout", "git_branch"},
			},
			{
				Command:      "git",
				Args:         []string{"pull", "git_address", "git_branch"},
				IsObfuscated: true,
				ErrorMsg:     "pull failed",
			},
		},
		expectedErr: "^Failed pull from branch: git_address/git_branch, error: pull failed$",
	})
}

func testClone(t *testing.T, tc simpleGitOperationTestCase) {
	testSingleOperation(t, tc, func(t build.Runner, s *gitSource) error { return s.clone(t) })
}

func testCheckout(t *testing.T, tc simpleGitOperationTestCase) {
	testSingleOperation(t, tc, func(t build.Runner, s *gitSource) error { return s.checkout(t) })
}

func testSingleOperation(t *testing.T, tc simpleGitOperationTestCase, funcToTest func(build.Runner, *gitSource) error) {
	source := NewGitSource(tc.address, tc.branch, tc.headRev, tc.targetDir, tc.cred)
	git := source.(*gitSource)
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation(tc.expectedCommands)
	defer runner.AssertExpectations(t)
	err := funcToTest(runner, git)
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile(tc.expectedErr), err.Error())
}

func TestCleanFailed(t *testing.T) {
	source := NewGitSource("address", "branch", "", "", nil)
	git := source.(*gitSource)
	runner := test.NewMockRunner()
	runner.PrepareCommandExpectation([]test.CommandsExpectation{
		{
			Command: "git",
			Args:    []string{"clean", "-xdf"},
		},
		{
			Command:  "git",
			Args:     []string{"reset", "--hard", "HEAD"},
			ErrorMsg: "permission denied",
		},
	})
	err := git.clean(runner)
	assert.NotNil(t, err)
	assert.Regexp(t, regexp.MustCompile("^Failed to discard local changes: permission denied$"), err.Error())
}
