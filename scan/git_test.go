// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License.

package scan

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	is "gotest.tools/v3/assert/cmp"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		doc      string
		url      string
		expected gitRepo
	}{
		{
			doc: "git scheme, no url-fragment",
			url: "git://github.com/user/repo.git",
			expected: gitRepo{
				remote: "git://github.com/user/repo.git",
				ref:    "",
			},
		},
		{
			doc: "git scheme, with url-fragment",
			url: "git://github.com/user/repo.git#mybranch:mydir/mysubdir/",
			expected: gitRepo{
				remote: "git://github.com/user/repo.git",
				ref:    "mybranch",
				subdir: "mydir/mysubdir/",
			},
		},
		{
			doc: "https scheme, no url-fragment",
			url: "https://github.com/user/repo.git",
			expected: gitRepo{
				remote: "https://github.com/user/repo.git",
				ref:    "",
			},
		},
		{
			doc: "https scheme, with url-fragment",
			url: "https://github.com/user/repo.git#mybranch:mydir/mysubdir/",
			expected: gitRepo{
				remote: "https://github.com/user/repo.git",
				ref:    "mybranch",
				subdir: "mydir/mysubdir/",
			},
		},
		{
			doc: "git@, no url-fragment",
			url: "git@github.com:user/repo.git",
			expected: gitRepo{
				remote: "git@github.com:user/repo.git",
				ref:    "",
			},
		},
		{
			doc: "git@, with url-fragment",
			url: "git@github.com:user/repo.git#mybranch:mydir/mysubdir/",
			expected: gitRepo{
				remote: "git@github.com:user/repo.git",
				ref:    "mybranch",
				subdir: "mydir/mysubdir/",
			},
		},
		{
			doc: "Azure DevOps Git URL, no url-fragment",
			url: "https://azure.visualstudio.com/ACR/_git/Build/",
			expected: gitRepo{
				remote: "https://azure.visualstudio.com/ACR/_git/Build/",
				ref:    "",
			},
		},
		{
			doc: "Azure DevOps Git URL, with url-fragment",
			url: "https://azure.visualstudio.com/ACR/_git/Build/#mybranch:mydir/mysubdir/",
			expected: gitRepo{
				remote: "https://azure.visualstudio.com/ACR/_git/Build/",
				ref:    "mybranch",
				subdir: "mydir/mysubdir/",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.doc, func(t *testing.T) {
			repo, err := parseRemoteURL(tc.url)
			assert.NilError(t, err)
			assert.Check(t, is.DeepEqual(tc.expected, repo, cmp.AllowUnexported(gitRepo{})))
		})
	}
}

func TestCloneArgsSmartHttp(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	serverURL, _ := url.Parse(server.URL)

	serverURL.Path = "/repo.git"

	mux.HandleFunc("/repo.git/info/refs", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("service")
		w.Header().Set("Content-Type", fmt.Sprintf("application/x-%s-advertisement", q))
	})

	args := fetchArgs(serverURL.String(), "main")
	exp := []string{"fetch", "--depth", "1", "origin", "main"}
	assert.Check(t, is.DeepEqual(exp, args))
}

func TestCloneArgsDumbHttp(t *testing.T) {
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	serverURL, _ := url.Parse(server.URL)

	serverURL.Path = "/repo.git"

	mux.HandleFunc("/repo.git/info/refs", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
	})

	args := fetchArgs(serverURL.String(), "main")
	exp := []string{"fetch", "origin", "main"}
	assert.Check(t, is.DeepEqual(exp, args))
}

func TestCloneArgsGit(t *testing.T) {
	args := fetchArgs("git://github.com/docker/docker", "main")
	exp := []string{"fetch", "--depth", "1", "origin", "main"}
	assert.Check(t, is.DeepEqual(exp, args))
}

func gitGetConfig(name string) string {
	b, err := git([]string{"config", "--get", name}...)
	if err != nil {
		// since we are interested in empty or non empty string,
		// we can safely ignore the err here.
		return ""
	}
	return strings.TrimSpace(string(b))
}

func TestCheckoutGit(t *testing.T) {
	root, err := ioutil.TempDir("", "docker-build-git-checkout")
	assert.NilError(t, err)
	defer os.RemoveAll(root)

	autocrlf := gitGetConfig("core.autocrlf")
	if !(autocrlf == "true" || autocrlf == "false" ||
		autocrlf == "input" || autocrlf == "") {
		t.Logf("unknown core.autocrlf value: \"%s\"", autocrlf)
	}
	eol := "\n"
	if autocrlf == "true" {
		eol = "\r\n"
	}

	gitDir := filepath.Join(root, "repo")
	_, err = git("init", gitDir)
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "config", "user.email", "test@docker.com")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "config", "user.name", "Docker test")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "checkout", "-b", "default")
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(gitDir, "Dockerfile"), []byte("FROM scratch"), 0600)
	assert.NilError(t, err)

	subDir := filepath.Join(gitDir, "subdir")
	assert.NilError(t, os.Mkdir(subDir, 0755))

	err = ioutil.WriteFile(filepath.Join(subDir, "Dockerfile"), []byte("FROM scratch\nEXPOSE 5000"), 0600)
	assert.NilError(t, err)

	if runtime.GOOS != "windows" {
		if err = os.Symlink("../subdir", filepath.Join(gitDir, "parentlink")); err != nil {
			t.Fatal(err)
		}

		if err = os.Symlink("/subdir", filepath.Join(gitDir, "absolutelink")); err != nil {
			t.Fatal(err)
		}
	}

	_, err = gitWithinDir(gitDir, "add", "-A")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "commit", "-am", "First commit")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "checkout", "-b", "test")
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(gitDir, "Dockerfile"), []byte("FROM scratch\nEXPOSE 3000"), 0600)
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(subDir, "Dockerfile"), []byte("FROM busybox\nEXPOSE 5000"), 0600)
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "add", "-A")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "commit", "-am", "Branch commit")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "checkout", "default")
	assert.NilError(t, err)

	// set up submodule
	subrepoDir := filepath.Join(root, "subrepo")
	_, err = git("init", subrepoDir)
	assert.NilError(t, err)

	_, err = gitWithinDir(subrepoDir, "config", "user.email", "test@docker.com")
	assert.NilError(t, err)

	_, err = gitWithinDir(subrepoDir, "config", "user.name", "Docker test")
	assert.NilError(t, err)

	err = ioutil.WriteFile(filepath.Join(subrepoDir, "subfile"), []byte("subcontents"), 0600)
	assert.NilError(t, err)

	_, err = gitWithinDir(subrepoDir, "add", "-A")
	assert.NilError(t, err)

	_, err = gitWithinDir(subrepoDir, "commit", "-am", "Subrepo initial")
	assert.NilError(t, err)

	cmd := exec.Command("git", "-c", "protocol.file.allow=always", "submodule", "add", subrepoDir, "sub") // this command doesn't work with --work-tree
	cmd.Dir = gitDir
	assert.NilError(t, cmd.Run())

	_, err = gitWithinDir(gitDir, "add", "-A")
	assert.NilError(t, err)

	_, err = gitWithinDir(gitDir, "commit", "-am", "With submodule")
	assert.NilError(t, err)

	type singleCase struct {
		frag      string
		exp       string
		fail      bool
		submodule bool
	}

	cases := []singleCase{
		{"", "FROM scratch", false, true},
		{"default", "FROM scratch", false, true},
		{":subdir", "FROM scratch" + eol + "EXPOSE 5000", false, false},
		{":nosubdir", "", true, false},   // missing directory error
		{":Dockerfile", "", true, false}, // not a directory error
		{"default:nosubdir", "", true, false},
		{"default:subdir", "FROM scratch" + eol + "EXPOSE 5000", false, false},
		{"default:../subdir", "", true, false},
		{"test", "FROM scratch" + eol + "EXPOSE 3000", false, false},
		{"test:", "FROM scratch" + eol + "EXPOSE 3000", false, false},
		{"test:subdir", "FROM busybox" + eol + "EXPOSE 5000", false, false},
		{"nonexist:subdir", "FROM busybox" + eol + "EXPOSE 5000", true, false},
	}

	if runtime.GOOS != "windows" {
		// Windows GIT (2.7.1 x64) does not support parentlink/absolutelink. Sample output below
		// 	git --work-tree .\repo --git-dir .\repo\.git add -A
		//	error: readlink("absolutelink"): Function not implemented
		// 	error: unable to index file absolutelink
		// 	fatal: adding files failed
		fmt.Println("Windows!!!!!!!!!!")
		cases = append(cases, singleCase{frag: "default:absolutelink", exp: "FROM scratch" + eol + "EXPOSE 5000", fail: false})
		cases = append(cases, singleCase{frag: "default:parentlink", exp: "FROM scratch" + eol + "EXPOSE 5000", fail: false})
	}

	for _, c := range cases {
		testroot, err := ioutil.TempDir("", "docker-build-git-checkout-test")
		assert.NilError(t, err)
		defer os.RemoveAll(testroot)

		ref, subdir := getRefAndSubdir(c.frag)
		r, err := cloneGitRepo(gitRepo{remote: gitDir, ref: ref, subdir: subdir}, testroot)

		if c.fail {
			assert.Check(t, is.ErrorContains(err, ""))
			continue
		}
		assert.NilError(t, err)
		defer os.RemoveAll(r)
		if c.submodule {
			b, innerErr := ioutil.ReadFile(filepath.Join(r, "sub/subfile"))
			assert.NilError(t, innerErr)
			assert.Check(t, is.Equal("subcontents", string(b)))
		} else {
			_, innerErr := os.Stat(filepath.Join(r, "sub/subfile"))
			assert.Assert(t, is.ErrorContains(innerErr, ""))
			assert.Assert(t, os.IsNotExist(innerErr))
		}

		b, err := ioutil.ReadFile(filepath.Join(r, "Dockerfile"))
		assert.NilError(t, err)
		assert.Check(t, is.Equal(c.exp, string(b)))
	}
}

func TestValidGitTransport(t *testing.T) {
	gitUrls := []string{
		"git://github.com/docker/docker",
		"git@github.com:docker/docker.git",
		"git@bitbucket.org:atlassianlabs/atlassian-docker.git",
		"https://github.com/docker/docker.git",
		"http://github.com/docker/docker.git",
		"http://github.com/docker/docker.git#branch",
		"http://github.com/docker/docker.git#:dir",
	}
	incompleteGitUrls := []string{
		"github.com/docker/docker",
	}

	for _, url := range gitUrls {
		if !isGitTransport(url) {
			t.Fatalf("%q should be detected as valid Git prefix", url)
		}
	}

	for _, url := range incompleteGitUrls {
		if isGitTransport(url) {
			t.Fatalf("%q should not be detected as valid Git prefix", url)
		}
	}
}

func TestSensorGitPAT(t *testing.T) {
	type message struct {
		before string
		after  string
	}

	messages := []message{
		{"fatal: Authentication failed for 'https://dummy:ABCabc123@github.com/username/reponame.git/'",
			"fatal: Authentication failed for 'https://<REDACTED>@github.com/username/reponame.git/'"},
	}

	for _, m := range messages {
		assert.Check(t, is.Equal(m.after, string(censorGitPAT([]byte(m.before)))))
	}
}
