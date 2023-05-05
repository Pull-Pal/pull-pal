package vc

import (
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mobyvb/pull-pal/llm"
	"go.uber.org/zap"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
)

// LocalGitClient represents a service that interacts with a local git repository.
type LocalGitClient struct {
	self Author
	repo Repository

	worktree *git.Worktree
}

// NewLocalGitClient initializes a local git client by checking out a repository locally.
func NewLocalGitClient( /*ctx context.Context, */ log *zap.Logger, self Author, repo Repository) (*LocalGitClient, error) {
	log.Info("checking out local github repo", zap.String("repo name", repo.Name), zap.String("local path", repo.LocalPath))
	// clone provided repository to local path
	if repo.LocalPath == "" {
		return nil, errors.New("local path to clone repository not provided")
	}

	// remove local repo if it exists already
	err := os.RemoveAll(repo.LocalPath)
	if err != nil {
		return nil, err
	}

	localRepo, err := git.PlainClone(repo.LocalPath, false, &git.CloneOptions{
		URL: repo.SSH(),
		// URL: repo.HTTPS(),
		Auth: &http.BasicAuth{
			Username: self.Handle,
			Password: self.Token,
		},
	})
	if err != nil {
		return nil, err
	}
	repo.localRepo = localRepo

	return &LocalGitClient{
		self: self,
		repo: repo,
	}, nil
}

/*
func (gc *LocalGitClient) SwitchBranch(branchName string) (err error) {
	if gc.worktree == nil {
		return errors.New("worktree is nil - cannot check out a branch")
	}

	branchRefName := plumbing.NewBranchReferenceName(branchName)
	// remoteName := "origin"

	err = gc.repo.localRepo.Fetch(&git.FetchOptions{
		RefSpecs: []config.RefSpec{"refs/*:refs/*", "HEAD:refs/heads/HEAD"},
	})
	if err != nil {
		return err
	}

	err = gc.worktree.Checkout(&git.CheckoutOptions{
		Branch: branchRefName,
		Force:  true,
	})
	if err != nil {
		return err
	}
		err = gc.repo.localRepo.CreateBranch(&config.Branch{
			Name:   branchName,
			Remote: remoteName,
			Merge:  branchRefName,
		})
		if err != nil {
			return err
		}

	return nil
}
*/

func (gc *LocalGitClient) PushBranch(branchName string) (err error) {
	//branchRefName := plumbing.NewBranchReferenceName(branchName)
	remoteName := "origin"

	// Push the new branch to the remote repository
	remote, err := gc.repo.localRepo.Remote(remoteName)
	if err != nil {
		return err
	}

	err = remote.Push(&git.PushOptions{
		RemoteName: remoteName,
		// TODO remove hardcoded "main"
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", "main", branchName))},
		Auth: &http.BasicAuth{
			Username: gc.self.Handle,
			Password: gc.self.Token,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (gc *LocalGitClient) GetLocalFile(path string) (llm.File, error) {
	fullPath := filepath.Join(gc.repo.LocalPath, path)

	data, err := ioutil.ReadFile(fullPath)
	if err != nil {
		// if file doesn't exist, just return an empty file
		// this means we want to prompt the llm to populate it for the first time
		if errors.Is(err, os.ErrNotExist) {
			return llm.File{
				Path:     path,
				Contents: "",
			}, nil
		}
		return llm.File{}, err
	}

	return llm.File{
		Path:     path,
		Contents: string(data),
	}, nil
}

func (gc *LocalGitClient) StartCommit() error {
	if gc.worktree != nil {
		return errors.New("worktree is not nil - cannot start a new commit")
	}

	worktree, err := gc.repo.localRepo.Worktree()
	if err != nil {
		return err
	}

	gc.worktree = worktree

	return nil
}

// ReplaceOrAddLocalFile updates or adds a file in the locally cloned repo, and applies these changes to the current git worktree.
func (gc *LocalGitClient) ReplaceOrAddLocalFile(newFile llm.File) error {
	if gc.worktree == nil {
		return errors.New("worktree is nil - StartCommit must be called")
	}

	// TODO format non-go files as well
	if strings.HasSuffix(newFile.Path, ".go") {
		newContents, err := format.Source([]byte(newFile.Contents))
		if err != nil {
			// TODO also make logger accessible
			fmt.Println("go format error")
			// TODO handle this error
			// return err
		} else {
			newFile.Contents = string(newContents)
		}
	}

	fullPath := filepath.Join(gc.repo.LocalPath, newFile.Path)

	err := ioutil.WriteFile(fullPath, []byte(newFile.Contents), 0644)
	if err != nil {
		return err
	}

	_, err = gc.worktree.Add(newFile.Path)

	return err
}

// FinishCommit completes a commit, after which a code change request can be opened or updated.
func (gc *LocalGitClient) FinishCommit(message string) error {
	if gc.worktree == nil {
		return errors.New("worktree is nil - StartCommit must be called")
	}
	_, err := gc.worktree.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  gc.self.Handle,
			Email: gc.self.Email,
			When:  time.Now(),
		},
	})
	if err != nil {
		return err
	}

	// set worktree to nil so a new commit can be started
	gc.worktree = nil

	return nil
}
