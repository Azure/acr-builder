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
var git = domain.Abstract("git")
var clone = domain.Abstract("clone")
var cloneBranch = domain.Abstract("-b")
var checkout = domain.Abstract("checkout")
var authTypePA = domain.Abstract("OAuth Personal Access Token")
var authTypeX = domain.Abstract("OAuth X Access Token")

type GitSource struct {
	Address       domain.AbstractString
	InitialBranch domain.AbstractString
	HeadRev       domain.AbstractString
	Credential    GitCredential
	TargetDir     domain.AbstractString
	stale         bool
}

func (s *GitSource) EnsureSource(runner domain.Runner) error {
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
		logrus.Infof("Directory %s exists, we will assume it's a valid git repository.", s.TargetDir.DisplayValue())
	} else {
		cloneArgs := []domain.AbstractString{*clone}
		if !s.InitialBranch.IsEmpty() {
			cloneArgs = append(cloneArgs, *cloneBranch, s.InitialBranch)
		}
		address, err := s.toAuthAddress(runner)
		if err != nil {
			return err
		}
		cloneArgs = append(cloneArgs, address, s.TargetDir)
		err = runner.ExecuteCmd(*git, cloneArgs)
		if err != nil {
			return fmt.Errorf("Error cloning git source: %s to directory %s", s.Address.DisplayValue(), s.TargetDir.DisplayValue())
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

func (s *GitSource) EnsureBranch(runner domain.Runner, branch domain.AbstractString) error {
	if branch.IsEmpty() {
		return fmt.Errorf("Branch parameter is required for git source")
	}
	defer func() { s.stale = true }()
	if s.stale {
		err := runner.ExecuteCmd(*git, []domain.AbstractString{*domain.Abstract("clean"), *domain.Abstract("-xdf")})
		if err != nil {
			return fmt.Errorf("Failed to clean repository: %s", err)
		}
		err = runner.ExecuteCmd(*git, []domain.AbstractString{*domain.Abstract("reset"), *domain.Abstract("--hard"), *domain.Abstract("HEAD")})
		if err != nil {
			return fmt.Errorf("Failed to discard local changes: %s", err)
		}
	}
	address, err := s.toAuthAddress(runner)
	if err != nil {
		return fmt.Errorf("Failed to get authorized address, error: %s", err)
	}
	if !s.HeadRev.IsEmpty() {
		if s.stale {
			err := runner.ExecuteCmd(*git, []domain.AbstractString{*domain.Abstract("fetch"), *domain.Abstract("--all")})
			if err != nil {
				return fmt.Errorf("Error fetching from: %s", address.DisplayValue())
			}
		}
		err = runner.ExecuteCmd(*git, []domain.AbstractString{*checkout, s.HeadRev})
		if err != nil {
			return fmt.Errorf("Error checking out revision: %s", s.HeadRev.DisplayValue())
		}
	} else {
		err := runner.ExecuteCmd(*git, []domain.AbstractString{*checkout, branch})
		if err != nil {
			return fmt.Errorf("Error checking out branch %s", branch.DisplayValue())
		}
		if s.stale {
			err = runner.ExecuteCmd(*git, []domain.AbstractString{*domain.Abstract("pull"), address, branch})
			if err != nil {
				return fmt.Errorf("Failed to pull from branch %s: %s", branch.DisplayValue(), err)
			}
		}
	}
	return nil
}

func (s *GitSource) Export() []domain.EnvVar {
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

func (s *GitSource) toAuthAddress(runner domain.Runner) (domain.AbstractString, error) {
	if s.Credential != nil {
		return s.Credential.toAuthAddress(runner, s.Address)
	}
	return s.Address, nil
}

type GitCredential interface {
	toAuthAddress(runner domain.Runner, address domain.AbstractString) (domain.AbstractString, error)
	Export() []domain.EnvVar
}

type GitPersonalAccessToken struct {
	user  domain.AbstractString
	token domain.AbstractString
}

func NewGitPersonalAccessToken(user string, token string) *GitPersonalAccessToken {
	return &GitPersonalAccessToken{
		user:  *domain.Abstract(user),
		token: *domain.AbstractSensitive(token),
	}
}

func (s *GitPersonalAccessToken) Export() []domain.EnvVar {
	return []domain.EnvVar{
		domain.EnvVar{
			Name:  constants.GitUserVar,
			Value: s.user,
		},
		domain.EnvVar{
			Name:  constants.GitAuthTypeVar,
			Value: *authTypePA,
		},
	}
}

func (s *GitPersonalAccessToken) toAuthAddress(runner domain.Runner, address domain.AbstractString) (domain.AbstractString, error) {
	userResolved := runner.Resolve(s.user)
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, userResolved+":"+tokenResolved)
}

type GitXToken struct {
	token domain.AbstractString
}

func NewXToken(token string) *GitXToken {
	return &GitXToken{token: *domain.AbstractSensitive(token)}
}

func (s *GitXToken) toAuthAddress(runner domain.Runner, address domain.AbstractString) (domain.AbstractString, error) {
	tokenResolved := runner.Resolve(s.token)
	return insertAuth(runner, address, "x-access-token:"+tokenResolved)
}

func (s *GitXToken) Export() []domain.EnvVar {
	return []domain.EnvVar{
		domain.EnvVar{
			Name:  constants.GitAuthTypeVar,
			Value: *authTypeX,
		},
	}
}

func insertAuth(runner domain.Runner, address domain.AbstractString, authString string) (domain.AbstractString, error) {
	addressResolved := runner.Resolve(address)
	protocolDivider := "://"
	if !strings.Contains(addressResolved, protocolDivider) {
		return domain.AbstractString{}, fmt.Errorf("Git repository address %s cannot be used with Personal Access Token", addressResolved)
	}
	addressAuthenticated := strings.Replace(addressResolved, protocolDivider, protocolDivider+authString+"@", 1)
	return *domain.AbstractSensitive(addressAuthenticated), nil
}
