package commands

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/Azure/acr-builder/pkg/domain"
	"github.com/sirupsen/logrus"
)

// Vocabulary to be used to build commands

type gitSource struct {
	Address       string
	InitialBranch string
	HeadRev       string
	Credential    GitCredential
	TargetDir     string
	stale         bool
}

// NewGitSource create a SourceDescription that represents a git checkout
func NewGitSource(address, initialBranch, headRev, targetDir string, credential GitCredential) domain.SourceDescription {
	return &gitSource{
		Address:       address,
		InitialBranch: initialBranch,
		HeadRev:       headRev,
		Credential:    credential,
		TargetDir:     targetDir,
	}
}

func (s *gitSource) EnsureSource(runner domain.Runner) error {
	if err := verifyGitVersion(); err != nil {
		logrus.Errorf("%s", err)
	}
	var targetEmpty bool
	targetExists, err := runner.DoesDirExist(s.TargetDir)
	if err != nil {
		return fmt.Errorf("Error checking for source dir: %s", err)
	}
	if targetExists {
		targetEmpty, err = runner.IsDirEmpty(s.TargetDir)
		if err != nil {
			return fmt.Errorf("Error checking if source dir is empty: %s", err)
		}
	}
	if targetExists && !targetEmpty {
		// target exists and not empty, we don't check out but will assume it's stale
		s.stale = true
		logrus.Infof("Directory %s exists, we will assume it's a valid git repository.", runner.Resolve(s.TargetDir))
	} else {
		cloneArgs := []string{"clone"}
		if s.InitialBranch != "" {
			cloneArgs = append(cloneArgs, "-b", s.InitialBranch)
		}
		address, err := s.toAuthAddress(runner)
		if err != nil {
			return err
		}
		cloneArgs = append(cloneArgs, address, s.TargetDir)
		err = runner.ExecuteCmd("git", cloneArgs)
		if err != nil {
			return fmt.Errorf("Error cloning git source: %s to directory %s", runner.Resolve(s.Address), runner.Resolve(s.TargetDir))
		}
	}

	err = runner.Chdir(s.TargetDir)
	if err != nil {
		return fmt.Errorf("Failed to chdir to %s", err)
	}
	return nil
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

func (s *gitSource) EnsureBranch(runner domain.Runner, branch string) error {
	if branch == "" {
		return fmt.Errorf("Branch parameter is required for git source")
	}
	defer func() { s.stale = true }()
	if s.stale {
		err := runner.ExecuteCmd("git", []string{"clean", "-xdf"})
		if err != nil {
			return fmt.Errorf("Failed to clean repository: %s", err)
		}
		err = runner.ExecuteCmd("git", []string{"reset", "--hard", "HEAD"})
		if err != nil {
			return fmt.Errorf("Failed to discard local changes: %s", err)
		}
	}
	address, err := s.toAuthAddress(runner)
	if err != nil {
		return fmt.Errorf("Failed to get authorized address, error: %s", err)
	}
	if s.HeadRev != "" {
		if s.stale {
			err := runner.ExecuteCmd("git", []string{"fetch", "--all"})
			if err != nil {
				return fmt.Errorf("Error fetching from: %s", runner.Resolve(address))
			}
		}
		err = runner.ExecuteCmd("git", []string{"checkout", s.HeadRev})
		if err != nil {
			return fmt.Errorf("Error checking out revision: %s", runner.Resolve(s.HeadRev))
		}
	} else {
		err := runner.ExecuteCmd("git", []string{"checkout", branch})
		if err != nil {
			return fmt.Errorf("Error checking out branch %s", runner.Resolve(branch))
		}
		if s.stale {
			err = runner.ExecuteCmd("git", []string{"pull", address, branch})
			if err != nil {
				return fmt.Errorf("Failed to pull from branch %s: %s", runner.Resolve(branch), err)
			}
		}
	}
	return nil
}

func (s *gitSource) Export() []domain.EnvVar {
	if s.Credential == nil {
		return []domain.EnvVar{}
	}
	credsExport := s.Credential.Export()
	return append(credsExport,
		domain.EnvVar{
			Name:  constants.GitSourceVar,
			Value: s.Address,
		},
		domain.EnvVar{
			Name:  constants.CheckoutDirVar,
			Value: s.TargetDir,
		},
	)
}

func (s *gitSource) toAuthAddress(runner domain.Runner) (string, error) {
	if s.Credential != nil {
		return s.Credential.toAuthAddress(runner, s.Address)
	}
	return s.Address, nil
}

// GitCredential objects are ways git authenticates
type GitCredential interface {
	toAuthAddress(runner domain.Runner, address string) (string, error)
	Export() []domain.EnvVar
}

type gitPersonalAccessToken struct {
	user  string
	token string
}

// NewGitPersonalAccessToken creates a GitCredential object with username and token/password
func NewGitPersonalAccessToken(user string, token string) GitCredential {
	return &gitPersonalAccessToken{
		user:  user,
		token: token,
	}
}

func (s *gitPersonalAccessToken) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.GitUserVar,
			Value: s.user,
		},
		{
			Name:  constants.GitAuthTypeVar,
			Value: "Git Personal Access Token",
		},
	}
}

func (s *gitPersonalAccessToken) toAuthAddress(runner domain.Runner, address string) (string, error) {
	userResolved := runner.Resolve(s.user)
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, userResolved+":"+tokenResolved)
}

type gitXToken struct {
	token string
}

// NewXToken creates a GitCredential object with a x-token
func NewXToken(token string) GitCredential {
	return &gitXToken{token: token}
}

func (s *gitXToken) toAuthAddress(runner domain.Runner, address string) (string, error) {
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, "x-access-token:"+tokenResolved)
}

func (s *gitXToken) Export() []domain.EnvVar {
	return []domain.EnvVar{
		{
			Name:  constants.GitAuthTypeVar,
			Value: "Git X Token",
		},
	}
}

func insertAuth(runner domain.Runner, address string, authString string) (string, error) {
	addressResolved := runner.Resolve(address)
	protocolDivider := "://"
	if !strings.Contains(addressResolved, protocolDivider) {
		return "", fmt.Errorf("Git repository address %s cannot be used with Personal Access Token", addressResolved)
	}
	addressAuthenticated := strings.Replace(addressResolved, protocolDivider, protocolDivider+authString+"@", 1)
	return addressAuthenticated, nil
}
