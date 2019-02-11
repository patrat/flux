package gittest

import (
	"context"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/weaveworks/flux"
	"github.com/weaveworks/flux/cluster/kubernetes/testfiles"
	"github.com/weaveworks/flux/git"
)

// Repo creates a new clone-able git repo, pre-populated with some kubernetes
// files and a few commits. Also returns a cleanup func to clean up after.
func Repo(t *testing.T) (*git.Repo, func()) {
	newDir, cleanup := testfiles.TempDir(t)

	filesDir := filepath.Join(newDir, "files")
	gitDir := filepath.Join(newDir, "git")
	if err := execCommand("mkdir", filesDir); err != nil {
		t.Fatal(err)
	}

	var err error
	if err = execCommand("git", "-C", filesDir, "init"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "config", "--local", "user.email", "example@example.com"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "config", "--local", "user.name", "example"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = testfiles.WriteTestFiles(filesDir); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "add", "--all"); err != nil {
		cleanup()
		t.Fatal(err)
	}
	if err = execCommand("git", "-C", filesDir, "commit", "-m", "'Initial revision'"); err != nil {
		cleanup()
		t.Fatal(err)
	}

	if err = execCommand("git", "clone", "--bare", filesDir, gitDir); err != nil {
		t.Fatal(err)
	}

	mirror := git.NewRepo(git.Remote{
		URL: "file://" + gitDir,
	})
	return mirror, func() {
		mirror.Clean()
		cleanup()
	}
}

// Workloads is a shortcut to getting the names of the workloads (NB
// not all resources, just the workloads) represented in the test
// files.
func Workloads() (res []flux.ResourceID) {
	for k, _ := range testfiles.ServiceMap("") {
		res = append(res, k)
	}
	return res
}

// CheckoutWithConfig makes a standard repo, clones it, and returns
// the clone, the original repo, and a cleanup function.
func CheckoutWithConfig(t *testing.T, config git.Config) (*git.Checkout, *git.Repo, func()) {
	repo, cleanup := Repo(t)
	if err := repo.Ready(context.Background()); err != nil {
		cleanup()
		t.Fatal(err)
	}

	co, err := repo.Clone(context.Background(), config)
	if err != nil {
		cleanup()
		t.Fatal(err)
	}
	return co, repo, func() {
		co.Clean()
		cleanup()
	}
}

var TestConfig git.Config = git.Config{
	Branch:    "master",
	UserName:  "example",
	UserEmail: "example@example.com",
	SyncTag:   "flux-test",
	NotesRef:  "fluxtest",
}

// Checkout makes a standard repo, clones it, and returns the clone
// with a cleanup function.
func Checkout(t *testing.T) (*git.Checkout, func()) {
	checkout, _, cleanup := CheckoutWithConfig(t, TestConfig)
	return checkout, cleanup
}

func execCommand(cmd string, args ...string) error {
	c := exec.Command(cmd, args...)
	c.Stderr = ioutil.Discard
	c.Stdout = ioutil.Discard
	return c.Run()
}
