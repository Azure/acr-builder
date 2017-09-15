package domain

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/Azure/acr-builder/pkg/constants"
	"github.com/sirupsen/logrus"
)

// Vocabulary to be used to build commands
var git = Abstract("git")
var clone = Abstract("clone")
var cloneBranch = Abstract("-b")
var checkout = Abstract("checkout")
var authTypePA = Abstract("OAuth Personal Access Token")
var authTypeX = Abstract("OAuth X Access Token")

type GitSource struct {
	Address       AbstractString
	InitialBranch AbstractString
	HeadRev       AbstractString
	Credential    GitCredential
	TargetDir     AbstractString
	stale         bool
}

func (s *GitSource) EnsureSource(runner Runner) error {
	if err := verifyGitVersion(); err != nil {
		logrus.Errorf("%s", err)
	}
	// TODO: try to clone with -b so we don't check out a branch we don't use
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
		logrus.Infof("Directory %s exists, we will assume it's a valid git repository.", s.TargetDir.value)
	} else {
		cloneArgs := []AbstractString{*clone}
		if s.InitialBranch.value != "" {
			cloneArgs = append(cloneArgs, *cloneBranch, s.InitialBranch)
		}
		address, err := s.toAuthAddress(runner)
		if err != nil {
			return err
		}
		cloneArgs = append(cloneArgs, address, s.TargetDir)
		err = runner.ExecuteCmd(*git, cloneArgs...)
		if err != nil {
			return fmt.Errorf("Error cloning git source: %s to directory %s", s.Address.value, s.TargetDir.value)
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

func (s *GitSource) EnsureBranch(runner Runner, branch AbstractString) error {
	if branch.value == "" {
		return fmt.Errorf("Branch parameter is required for git source")
	}
	defer func() { s.stale = true }()
	if s.stale {
		err := runner.ExecuteCmd(*git, *Abstract("clean"), *Abstract("-xdf"))
		if err != nil {
			return fmt.Errorf("Failed to clean repository: %s", err)
		}
		err = runner.ExecuteCmd(*git, *Abstract("reset"), *Abstract("--hard"), *Abstract("HEAD"))
		if err != nil {
			return fmt.Errorf("Failed to discard local changes: %s", err)
		}
	}
	address, err := s.toAuthAddress(runner)
	if err != nil {
		return fmt.Errorf("Failed to get authorized address, error: %s", err)
	}
	if s.HeadRev.value != "" {
		if s.stale {
			err := runner.ExecuteCmd(*git, *Abstract("fetch"), address)
			if err != nil {
				return fmt.Errorf("Error fetching from: %s", address.value)
			}
		}
		err = runner.ExecuteCmd(*git, *checkout, s.HeadRev)
		if err != nil {
			return fmt.Errorf("Error checking out revision: %s", s.HeadRev.value)
		}
	} else {
		err := runner.ExecuteCmd(*git, *checkout, branch)
		if err != nil {
			return fmt.Errorf("Error checking out branch %s", branch.value)
		}
		if s.stale {
			err = runner.ExecuteCmd(*git, *Abstract("pull"), address, branch)
			if err != nil {
				return fmt.Errorf("Failed to pull from branch %s: %s", branch.value, err)
			}
		}
	}
	return nil
}

func (s *GitSource) Export() []EnvVar {
	if s.Credential == nil {
		return []EnvVar{}
	}
	credsExport := s.Credential.Export()
	return append(credsExport,
		EnvVar{
			Name:  constants.GitSourceVar,
			Value: s.Address,
		},
		EnvVar{
			Name:  constants.CheckoutDirVar,
			Value: s.TargetDir,
		},
	)
}

func (s *GitSource) toAuthAddress(runner Runner) (AbstractString, error) {
	if s.Credential != nil {
		return s.Credential.toAuthAddress(runner, s.Address)
	}
	return s.Address, nil
}

type GitCredential interface {
	toAuthAddress(runner Runner, address AbstractString) (AbstractString, error)
	Export() []EnvVar
}

type GitPersonalAccessToken struct {
	user  AbstractString
	token AbstractString
}

func NewGitPersonalAccessToken(user string, token string) *GitPersonalAccessToken {
	return &GitPersonalAccessToken{
		user:  *Abstract(user),
		token: *AbstractSensitive(token),
	}
}

func (s *GitPersonalAccessToken) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			constants.GitUserVar,
			s.user,
		},
		EnvVar{
			constants.GitAuthTypeVar,
			*authTypePA,
		},
	}
}

func (s *GitPersonalAccessToken) toAuthAddress(runner Runner, address AbstractString) (AbstractString, error) {
	userResolved := runner.Resolve(s.user)
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, userResolved+":"+tokenResolved)
}

type GitXToken struct {
	token AbstractString
}

func NewXToken(token string) *GitXToken {
	return &GitXToken{token: *AbstractSensitive(token)}
}

func (s *GitXToken) toAuthAddress(runner Runner, address AbstractString) (AbstractString, error) {
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, "x-access-token:"+tokenResolved)
}

func (s *GitXToken) Export() []EnvVar {
	return []EnvVar{
		EnvVar{
			constants.GitAuthTypeVar,
			*authTypeX,
		},
	}
}

func insertAuth(runner Runner, address AbstractString, authString string) (AbstractString, error) {
	addressResolved := runner.Resolve(address)
	protocolDivider := "://"
	if !strings.Contains(addressResolved, protocolDivider) {
		return AbstractString{}, fmt.Errorf("Git repository address %s cannot be used with Personal Access Token", addressResolved)
	}
	addressAuthenticated := strings.Replace(addressResolved, protocolDivider, protocolDivider+authString+"@", 1)
	return *AbstractSensitive(addressAuthenticated), nil
}
