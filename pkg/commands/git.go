package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	build "github.com/Azure/acr-builder/pkg"
	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/sirupsen/logrus"
)

const defaultTargetDir = "src" // force targetDir to not be null so we never clone against .

type gitSource struct {
	address    string
	branch     string
	headRev    string
	credential GitCredential
	targetDir  string
	tracker    *DirectoryTracker
}

// NewGitSource create a SourceDescription that represents a git checkout
func NewGitSource(address, branch, headRev, targetDir string, credential GitCredential) build.Source {
	if targetDir == "" {
		targetDir = defaultTargetDir
	}
	return &gitSource{
		address:    address,
		branch:     branch,
		headRev:    headRev,
		credential: credential,
		targetDir:  targetDir,
	}
}

func (s *gitSource) Return(runner build.Runner) error {
	if s.tracker != nil {
		return s.tracker.Return(runner)
	}
	return nil
}

func (s *gitSource) Obtain(runner build.Runner) error {
	if err := verifyGitVersion(); err != nil {
		logrus.Errorf("Error verifying Git version: %s", err)
	}
	env := runner.GetContext()
	if exists, err := s.targetExists(runner); err != nil {
		return err
	} else if exists {
		logrus.Debugf("Directory %s exists, we will assume it's a valid git repository.", env.Expand(s.targetDir))
		tracker, err := ChdirWithTracking(runner, s.targetDir)
		if err != nil {
			return err
		}
		s.tracker = tracker

		if err := s.clean(runner); err != nil {
			return err
		}
		if err := s.checkout(runner); err != nil {
			return err
		}
	} else {
		if err := s.clone(runner); err != nil {
			return err
		}
		tracker, err := ChdirWithTracking(runner, s.targetDir)
		if err != nil {
			return err
		}
		s.tracker = tracker
		if s.headRev != "" {
			err := s.ensureHeadRev(runner)
			if err != nil {
				return err
			}
		}
	}

	// NOTE: We need to export the GitHeadRev through os.env as Source.Export is called before Source.Obtain
	// It's a limitation in Workflow engine and we need to review the design.
	return s.exportGitHeadRev(runner)
}

func (s *gitSource) exportGitHeadRev(runner build.Runner) error {
	rev, err := s.getHeadRev(runner)
	if err == nil {
		err = os.Setenv(constants.ExportsGitHeadRev, strings.TrimSpace(rev))
	}
	return err
}

func (s *gitSource) targetExists(runner build.Runner) (bool, error) {
	env := runner.GetContext()
	fs := runner.GetFileSystem()
	var targetDir string
	if s.targetDir != "" {
		targetDir = s.targetDir
		dirExists, err := fs.DoesDirExist(targetDir)
		if err != nil {
			return false, fmt.Errorf("Error checking for source dir: %s, error: %s", env.Expand(targetDir), err)
		}
		if !dirExists {
			return false, nil
		}
	} else {
		targetDir = "."
	}

	dirEmpty, err := fs.IsDirEmpty(targetDir)
	if err != nil {
		return false, fmt.Errorf("Error checking if source dir is empty: %s, error: %s", env.Expand(targetDir), err)
	}
	return !dirEmpty, nil
}

func (s *gitSource) clone(runner build.Runner) error {
	env := runner.GetContext()
	args := []string{"clone"}
	if s.branch != "" {
		args = append(args, "-b", s.branch)
	}
	remote, obfuscated, err := s.toAuthAddress(runner)
	if err != nil {
		return fmt.Errorf("Failed to get authorized address, error: %s", err)
	}
	obfuscator := gitAddressObfuscator(remote, obfuscated)
	args = append(args, remote)
	if s.targetDir != "" {
		args = append(args, s.targetDir)
	}
	err = runner.ExecuteCmdWithObfuscation(obfuscator, "git", args)
	if err != nil {
		var target string
		if s.targetDir == "" {
			target = "<current directory>"
		} else {
			target = env.Expand(s.targetDir)
		}
		return fmt.Errorf("Error cloning git source: %s to directory %s, error: %s", obfuscated, target, err)
	}
	return nil
}

func (s *gitSource) clean(runner build.Runner) error {
	if err := runner.ExecuteCmd("git", []string{"clean", "-xdf"}, nil); err != nil {
		return fmt.Errorf("Failed to clean repository: %s", err)
	}
	// reset shouldn't be necessary...
	if err := runner.ExecuteCmd("git", []string{"reset", "--hard", "HEAD"}, nil); err != nil {
		return fmt.Errorf("Failed to discard local changes: %s", err)
	}
	return nil
}

func (s *gitSource) checkout(runner build.Runner) error {
	env := runner.GetContext()
	remote, obfuscated, err := s.toAuthAddress(runner)
	if err != nil {
		return fmt.Errorf("Failed to get authorized address, error: %s", err)
	}
	obfuscator := gitAddressObfuscator(remote, obfuscated)
	if err := runner.ExecuteCmdWithObfuscation(obfuscator, "git", []string{"fetch", remote}); err != nil {
		return fmt.Errorf("Failed to clean fetch from remote: %s, error: %s", obfuscated, err)
	}

	if s.headRev != "" {
		return s.ensureHeadRev(runner)
	}

	if s.branch != "" {
		if err := s.checkoutAt(runner, s.branch); err != nil {
			return err
		}
	}

	pullArgs := []string{"pull", remote}
	if s.branch != "" {
		pullArgs = append(pullArgs, s.branch)
	}
	if err := runner.ExecuteCmdWithObfuscation(obfuscator, "git", pullArgs); err != nil {
		return fmt.Errorf("Failed pull from branch: %s/%s, error: %s", obfuscated, env.Expand(s.branch), err)
	}

	return nil
}

func (s *gitSource) ensureHeadRev(runner build.Runner) error {
	env := runner.GetContext()
	if s.branch != "" {
		logrus.Debugf("Ignoring branch %s since head rev %s is given...", env.Expand(s.branch), env.Expand(s.headRev))
	}
	return s.checkoutAt(runner, s.headRev)
}

func (s *gitSource) checkoutAt(runner build.Runner, checkoutTarget string) error {
	env := runner.GetContext()
	if err := runner.ExecuteCmd("git", []string{"checkout", checkoutTarget}, nil); err != nil {
		return fmt.Errorf("Failed checkout git repository at: %s, error: %s", env.Expand(checkoutTarget), err)
	}
	return nil
}

func (s *gitSource) getHeadRev(runner build.Runner) (string, error) {
	return runner.QueryCmd("git", []string{"rev-parse", "--verify", "HEAD"})
}

func verifyGitVersion() error {
	// NOTE: Git 2.13 is known to have a major security vulnerability
	gitVerifyCmd := exec.Command("git", "--version")
	gitVersionString, err := gitVerifyCmd.Output()
	if err != nil {
		return fmt.Errorf("Failed to verify git command, %s", err)
	}
	gitVersionTokens := strings.Fields(string(gitVersionString))
	gitVersionNumber := gitVersionTokens[len(gitVersionTokens)-1]
	gitVersionNumberTokens := strings.Split(gitVersionNumber, ".")
	if len(gitVersionNumberTokens) < 2 {
		return fmt.Errorf("Unexpected git version number: %s", gitVersionNumber)
	}
	gitMajorVersion, err := strconv.Atoi(gitVersionNumberTokens[0])
	if err != nil {
		return fmt.Errorf("Unexpected git version number: %s", gitVersionNumber)
	}
	gitMinorVersion, err := strconv.Atoi(gitVersionNumberTokens[1])
	if err != nil {
		return fmt.Errorf("Unexpected git version number: %s", gitVersionNumber)
	}
	if gitMajorVersion < 2 || gitMinorVersion < 14 {
		return fmt.Errorf("Please consider using Git version 2.14.0 or higher")
	}
	return nil
}

func gitAddressObfuscator(address, obfuscation string) func(args []string) {
	return func(args []string) {
		replaced := false
		for i := 0; i < len(args); i++ {
			if args[i] == address {
				args[i] = obfuscation
				replaced = true
			}
		}
		if !replaced {
			logrus.Errorf("Unexpectedly unable to obfuscate git address")
		}
	}
}

func (s *gitSource) Export() []build.EnvVar {
	var exports []build.EnvVar
	if s.credential != nil {
		exports = s.credential.Export()
	} else {
		exports = []build.EnvVar{}
	}
	if s.targetDir != "" {
		exports = append(exports, build.EnvVar{
			Name:  constants.ExportsWorkingDir,
			Value: s.targetDir,
		})
	}
	if s.branch != "" {
		exports = append(exports, build.EnvVar{
			Name:  constants.ExportsGitBranch,
			Value: s.branch,
		})
	}
	return append(exports,
		build.EnvVar{
			Name:  constants.ExportsGitSource,
			Value: s.address,
		},
	)
}

func (s *gitSource) toAuthAddress(runner build.Runner) (string, string, error) {
	if s.credential != nil {
		return s.credential.toAuthAddress(runner, s.address)
	}
	return s.address, s.address, nil
}

// GitCredential objects are ways git authenticates
type GitCredential interface {
	toAuthAddress(runner build.Runner, address string) (string, string, error)
	Export() []build.EnvVar
}

type gitPersonalAccessToken struct {
	user  string
	token string
}

// NewGitPersonalAccessToken creates a GitCredential object with username and token/password
func NewGitPersonalAccessToken(user string, token string) (GitCredential, error) {
	if (user == "") != (token == "") {
		return nil, fmt.Errorf("Please provide both --%s and --%s or neither", constants.ArgNameGitPATokenUser, constants.ArgNameGitPAToken)
	}
	return &gitPersonalAccessToken{
		user:  user,
		token: token,
	}, nil
}

func (s *gitPersonalAccessToken) Export() []build.EnvVar {
	return []build.EnvVar{
		{
			Name:  constants.ExportsGitUser,
			Value: s.user,
		},
		{
			Name:  constants.ExportsGitAuthType,
			Value: "Git Personal Access Token",
		},
	}
}

func (s *gitPersonalAccessToken) toAuthAddress(runner build.Runner, address string) (authAddress, obfuscation string, err error) {
	env := runner.GetContext()
	userResolved := env.Expand(s.user)
	tokenResolved := env.Expand(s.token)
	authAddress, err = insertAuth(runner, address, userResolved+":"+tokenResolved)
	if err != nil {
		return
	}
	obfuscation, err = insertAuth(runner, address, userResolved+":"+constants.ObfuscationString)
	return
}

type gitXToken struct {
	token string
}

// NewGitXToken creates a GitCredential object with a x-token
func NewGitXToken(token string) GitCredential {
	return &gitXToken{token: token}
}

func (s *gitXToken) toAuthAddress(runner build.Runner, address string) (authAddress, obfuscation string, err error) {
	env := runner.GetContext()
	tokenResolved := env.Expand(s.token)
	authAddress, err = insertAuth(runner, address, "x-access-token:"+tokenResolved)
	if err != nil {
		return
	}
	obfuscation, err = insertAuth(runner, address, "x-access-token:"+constants.ObfuscationString)
	return
}

func (s *gitXToken) Export() []build.EnvVar {
	return []build.EnvVar{
		{
			Name:  constants.ExportsGitAuthType,
			Value: "Git X Token",
		},
	}
}

func insertAuth(runner build.Runner, address string, authString string) (string, error) {
	env := runner.GetContext()
	addressResolved := env.Expand(address)
	protocolDivider := "://"
	if !strings.Contains(addressResolved, protocolDivider) {
		return "", fmt.Errorf("Git repository address %s cannot be used with Access Tokens", addressResolved)
	}
	addressAuthenticated := strings.Replace(addressResolved, protocolDivider, protocolDivider+authString+"@", 1)
	return addressAuthenticated, nil
}
