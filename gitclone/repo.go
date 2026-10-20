package gitclone

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

func Ensure(repoURL, branch, dir string) error {
	branchRef := plumbing.NewBranchReferenceName(branch)

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find user home directory: %w", err)
	}
	auth, err := ssh.NewPublicKeysFromFile("git", filepath.Join(homeDir, ".ssh", "id_ed25519"), "")
	if err != nil {
		return fmt.Errorf("load SSH private key: %w", err)
	}

	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		fmt.Printf("Cloning %s (%s) -> %s\n", repoURL, branch, dir)
		_, err := git.PlainClone(dir, false, &git.CloneOptions{
			URL:           repoURL,
			Auth:          auth,
			ReferenceName: branchRef,
			SingleBranch:  true,
			Depth:         1,
			Progress:      os.Stdout,
		})
		return err
	} else if err != nil {
		return fmt.Errorf("check repository directory: %w", err)
	}

	fmt.Printf("Updating %s (%s)\n", dir, branch)

	r, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("folder exists but not a git repo: %w", err)
	}

	cfg, err := r.Config()
	if err != nil {
		return fmt.Errorf("read repository config: %w", err)
	}
	remote, exists := cfg.Remotes["origin"]
	if !exists {
		_, err = r.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{repoURL},
		})
	} else {
		remote.URLs = []string{repoURL}
		err = r.SetConfig(cfg)
	}
	if err != nil {
		return fmt.Errorf("configure origin remote: %w", err)
	}

	err = r.Fetch(&git.FetchOptions{
		Auth:     auth,
		Progress: os.Stdout,
		RefSpecs: []config.RefSpec{
			"+refs/heads/*:refs/remotes/origin/*",
			"+refs/tags/*:refs/tags/*",
		},
		Force: true,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("fetch failed: %w", err)
	}

	wt, err := r.Worktree()
	if err != nil {
		return err
	}

	_ = wt.Checkout(&git.CheckoutOptions{
		Branch: branchRef,
		Force:  true,
	})
	remoteRef := plumbing.NewRemoteReferenceName("origin", branch)
	if ref, rerr := r.Reference(remoteRef, true); rerr == nil {
		_ = wt.Checkout(&git.CheckoutOptions{
			Hash:   ref.Hash(),
			Branch: branchRef,
			Create: true,
			Force:  true,
		})
	}

	err = wt.Pull(&git.PullOptions{
		Auth:          auth,
		ReferenceName: branchRef,
		SingleBranch:  true,
		Force:         true,
		Progress:      os.Stdout,
	})
	if err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		return fmt.Errorf("pull failed: %w", err)
	}
	return nil
}
