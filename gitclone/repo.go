package gitclone

import (
	"errors"
	"fmt"
	"os"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

const EmbeddedGitToken = "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"

func Ensure(repoURL, branch, dir string) error {
	branchRef := plumbing.NewBranchReferenceName(branch)

	var auth *http.BasicAuth

	if EmbeddedGitToken != "" {
		auth = &http.BasicAuth{
			Username: "x-access-token",
			Password: EmbeddedGitToken,
		}
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
	}

	fmt.Printf("Updating %s (%s)\n", dir, branch)

	r, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("folder exists but not a git repo: %w", err)
	}

	// fetch origin refs
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

	// ensure branch checked out
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
