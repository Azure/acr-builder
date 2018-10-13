// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/docker/docker/pkg/symlink"
	"github.com/docker/docker/pkg/urlutil"
	"github.com/pkg/errors"
)

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
type gitRepo struct {
	remote string
	ref    string
	subdir string
}

// GetGitCommitID queries git for the latest commit.
func (s *Scanner) GetGitCommitID(ctx context.Context, cmdDir string) (string, error) {
	cmd := []string{"git", "rev-parse", "--verify", "HEAD"}
	var buf bytes.Buffer
	if err := s.procManager.Run(ctx, cmd, nil, &buf, os.Stderr, cmdDir); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}

// Clone clones a repository into a newly created directory, returning the resulting directory name.
func Clone(remoteURL string, root string) (string, error) {
	repo, err := parseRemoteURL(remoteURL)
	if err != nil {
		return "", err
	}

	return cloneGitRepo(repo, root)
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func cloneGitRepo(repo gitRepo, root string) (checkoutDir string, err error) {
	fetch := fetchArgs(repo.remote, repo.ref)
	if err != nil {
		return "", err
	}

	defer func() {
		if err != nil {
			_ = os.RemoveAll(root)
		}
	}()

	if out, err := gitWithinDir(root, "init"); err != nil {
		return "", errors.Wrapf(err, "failed to init repo at %s: %s", root, out)
	}

	// Add origin remote for compatibility with previous implementation that
	// used "git clone" and also to make sure local refs are created for branches
	if out, err := gitWithinDir(root, "remote", "add", "origin", repo.remote); err != nil {
		return "", errors.Wrapf(err, "failed add origin repo at %s: %s", repo.remote, out)
	}

	if _, err := gitWithinDir(root, fetch...); err != nil {
		// Fall back to full fetch if shallow fetch fails.
		// It's mainly for the scenario if the reference is a git commit,
		// eg, https://github.com/abc.git#bcaf8913695e5ad57868c8c82af58f9e699e7f59
		if output2, err2 := gitWithinDir(root, "fetch", "origin"); err2 != nil {
			return "", errors.Wrapf(err, "error fetching: %s", output2)
		}
	}

	checkoutDir, err = checkoutGit(root, repo.ref, repo.subdir)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("git", "submodule", "update", "--init", "--recursive", "--depth=1")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "error initializing submodules: %s", output)
	}

	err = gitLfs(root)
	if err != nil {
		return "", err
	}

	return checkoutDir, nil
}

func gitLfs(root string) error {
	err := exec.Command("git-lfs", "version").Run()
	if err == nil {
		cmd := exec.Command("git", "lfs", "pull")
		cmd.Dir = root

		output, err := cmd.CombinedOutput()
		if err != nil {
			return errors.Wrapf(err, "error executing 'git lfs pull': %s", output)
		}
	} else {
		log.Println("WARNING: git-lfs is not installed")
	}

	return nil
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func checkoutGit(root, ref, subdir string) (string, error) {
	// Try checking out by ref name first. This will work on branches and sets
	// .git/HEAD to the current branch name
	if output, err := gitWithinDir(root, "checkout", ref); err != nil {
		// If checking out by branch name fails check out the last fetched ref
		if _, err2 := gitWithinDir(root, "checkout", "FETCH_HEAD"); err2 != nil {
			return "", errors.Wrapf(err, "error checking out %s: %s", ref, output)
		}
	}

	if subdir != "" {
		newCtx, err := symlink.FollowSymlinkInScope(filepath.Join(root, subdir), root)
		if err != nil {
			return "", errors.Wrapf(err, "error setting git context, %q not within git root", subdir)
		}

		fi, err := os.Stat(newCtx)
		if err != nil {
			return "", err
		}
		if !fi.IsDir() {
			return "", errors.Errorf("error setting git context, not a directory: %s", newCtx)
		}
		root = newCtx
	}

	return root, nil
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func parseRemoteURL(remoteURL string) (gitRepo, error) {
	repo := gitRepo{}

	if !isGitTransport(remoteURL) {
		remoteURL = "https://" + remoteURL
	}

	var fragment string
	if strings.HasPrefix(remoteURL, "git@") {
		// git@.. is not an URL, so cannot be parsed as URL
		parts := strings.SplitN(remoteURL, "#", 2)

		repo.remote = parts[0]
		if len(parts) == 2 {
			fragment = parts[1]
		}
		repo.ref, repo.subdir = getRefAndSubdir(fragment)
	} else {
		u, err := url.Parse(remoteURL)
		if err != nil {
			return repo, err
		}

		repo.ref, repo.subdir = getRefAndSubdir(u.Fragment)
		u.Fragment = ""
		repo.remote = u.String()
	}
	return repo, nil
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func fetchArgs(remoteURL string, ref string) []string {
	args := []string{"fetch"}

	if supportsShallowClone(remoteURL) {
		args = append(args, "--depth", "1")
	}

	return append(args, "origin", ref)
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func getRefAndSubdir(fragment string) (ref string, subdir string) {
	refAndDir := strings.SplitN(fragment, ":", 2)
	ref = "master"
	if len(refAndDir[0]) != 0 {
		ref = refAndDir[0]
	}
	if len(refAndDir) > 1 && len(refAndDir[1]) != 0 {
		subdir = refAndDir[1]
	}
	return
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func supportsShallowClone(remoteURL string) bool {
	if urlutil.IsURL(remoteURL) {
		// Check if the HTTP server is smart

		// Smart servers must correctly respond to a query for the git-upload-pack service
		serviceURL := remoteURL + "/info/refs?service=git-upload-pack"

		// Try a HEAD request and fallback to a Get request on error
		res, err := http.Head(serviceURL)
		if err != nil || res.StatusCode != http.StatusOK {
			res, err = http.Get(serviceURL)
			if err == nil {
				_ = res.Body.Close()
			}
			if err != nil || res.StatusCode != http.StatusOK {
				// request failed
				return false
			}
		}

		if res.Header.Get("Content-Type") != "application/x-git-upload-pack-advertisement" {
			// Fallback, not a smart server
			return false
		}
		return true
	}

	// Non-HTTP protocols always support shallow clones
	return true
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func gitWithinDir(dir string, args ...string) ([]byte, error) {
	a := []string{"--work-tree", dir, "--git-dir", filepath.Join(dir, ".git")}
	return git(append(a, args...)...)
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func git(args ...string) ([]byte, error) {
	return exec.Command("git", args...).CombinedOutput()
}

// ref: https://github.com/moby/moby/blob/master/builder/remotecontext/git/gitutils.go
func isGitTransport(str string) bool {
	return urlutil.IsURL(str) || strings.HasPrefix(str, "git://") || strings.HasPrefix(str, "git@")
}
