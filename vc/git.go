package vc

import (
	"errors"
	"fmt"
	"go/format"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/mobyvb/pull-pal/llm"
	"go.uber.org/zap"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// LocalGitClient represents a service that interacts with a local git repository.
type LocalGitClient struct {
	log  *zap.Logger
	self Author
	repo Repository

	worktree      *git.Worktree
	debugDir      string
	cloneProtocol string
}

// NewLocalGitClient initializes a local git client by checking out a repository locally.
func NewLocalGitClient(log *zap.Logger, self Author, repo Repository, debugDir string, cloneProtocol string) (*LocalGitClient, error) {
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

	auth := &http.BasicAuth{
		Username: self.Handle,
		Password: self.Token,
	}

	if cloneProtocol == "SSH" {
		auth = &ssh.PublicKeys{User: "git", Signer: self.SSHKey}
	}

	localRepo, err := git.PlainClone(repo.LocalPath, false, &git.CloneOptions{
		URL:  repo.CloneURL(cloneProtocol),
		Auth: auth,
	})
	if err != nil {
		return nil, err
	}
	repo.localRepo = localRepo

	return &LocalGitClient{
		log:           log,
		self:          self,
		repo:          repo,
		debugDir:      debugDir,
		cloneProtocol: cloneProtocol,
	}, nil
}

// [remainder of file...]
