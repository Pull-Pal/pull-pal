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
)

// LocalGitClient represents a service that interacts with a local git repository.
type LocalGitClient struct {
	log  *zap.Logger
	self Author
	repo Repository

	worktree *git.Worktree
	debugDir string
}

// NewLocalGitClient initializes a local git client by checking out a repository locally.
func NewLocalGitClient(log *zap.Logger, self Author, repo Repository, debugDir string) (*LocalGitClient, error) {
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
		log:      log,
		self:     self,
		repo:     repo,
		debugDir: debugDir,
	}, nil
}

func (gc *LocalGitClient) CheckoutRemoteBranch(branchName string) (err error) {
	if gc.worktree == nil {
		return errors.New("worktree is nil - cannot check out a branch")
	}

	// TODO configurable remote
	branchRefName := plumbing.NewRemoteReferenceName("origin", branchName)
	branchCoOpts := git.CheckoutOptions{
		Branch: plumbing.ReferenceName(branchRefName),
		Force:  true,
	}
	err = gc.worktree.Checkout(&branchCoOpts)
	if err != nil {
		return err
	}

	// Pull the latest changes from the remote branch
	/*
		err = gc.worktree.Pull(&git.PullOptions{
			RemoteName: "origin",
			Auth: &http.BasicAuth{
				Username: gc.self.Handle,
				Password: gc.self.Token,
			},
			Force: true,
		})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			return err
		}
	*/
	return nil
}

func (gc *LocalGitClient) PushBranch(branchName string) (err error) {
	//branchRefName := plumbing.NewBranchReferenceName(branchName)
	remoteName := "origin"

	headRef, err := gc.repo.localRepo.Head()
	if err != nil {
		return err
	}

	// Create new branch at current HEAD
	branchRef := plumbing.NewHashReference(plumbing.NewBranchReferenceName(branchName), headRef.Hash())
	err = gc.repo.localRepo.Storer.SetReference(branchRef)
	if err != nil {
		return err
	}

	// Push the new branch to the remote repository
	remote, err := gc.repo.localRepo.Remote(remoteName)
	if err != nil {
		return err
	}

	err = remote.Push(&git.PushOptions{
		RemoteName: remoteName,
		// TODO remove hardcoded "main"
		RefSpecs: []config.RefSpec{config.RefSpec(fmt.Sprintf("+refs/heads/%s:refs/heads/%s", branchName, branchName))},
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
	dirPath := filepath.Dir(fullPath)
	err := os.MkdirAll(dirPath, 0755)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(fullPath, []byte(newFile.Contents), 0644)
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

// ParseIssueAndStartCommit parses the information provided in the issue to check out the appropriate branch,
// get the contents of the files mentioned in the issue, and initialize the worktree.
func (gc *LocalGitClient) ParseIssueAndStartCommit(issue Issue) (llm.CodeChangeRequest, error) {
	var changeRequest llm.CodeChangeRequest

	if gc.worktree != nil {
		return changeRequest, errors.New("worktree is active - some other work is incomplete")
	}

	issueBody := ParseIssueBody(issue.Body)
	gc.log.Info("issue body info", zap.Any("files", issueBody.FilePaths))

	// start a worktree
	err := gc.StartCommit()
	if err != nil {
		gc.log.Error("error starting commit", zap.Error(err))
		return changeRequest, err
	}

	err = gc.CheckoutRemoteBranch(issueBody.BaseBranch)
	if err != nil {
		gc.log.Error("error checking out remote branch", zap.Error(err))
		return changeRequest, err
	}

	// get file contents from local git repository
	files := []llm.File{}
	for _, path := range issueBody.FilePaths {
		nextFile, err := gc.GetLocalFile(path)
		if err != nil {
			gc.log.Error("error getting local file", zap.Error(err))
			return changeRequest, err
		}
		files = append(files, nextFile)
	}

	req := llm.CodeChangeRequest{
		Subject:     issue.Subject,
		Body:        issueBody.PromptBody,
		IssueNumber: issue.Number,
		Files:       files,
		BaseBranch:  issueBody.BaseBranch,
	}
	debugFileNamePrefix := fmt.Sprintf("issue-%d-%d", issue.Number, time.Now().Unix())
	gc.writeDebug("issues", debugFileNamePrefix+"-originalbody.txt", issue.Body)
	gc.writeDebug("issues", debugFileNamePrefix+"-parsed-req.txt", req.String())

	return req, nil
}

func (gc *LocalGitClient) writeDebug(subdir, filename, contents string) {
	if gc.debugDir == "" {
		return
	}

	fullFolderPath := path.Join(gc.debugDir, subdir)

	err := os.MkdirAll(fullFolderPath, os.ModePerm)
	if err != nil {
		gc.log.Error("failed to ensure debug directory existed", zap.String("folderpath", fullFolderPath), zap.Error(err))
		return
	}

	fullPath := path.Join(fullFolderPath, filename)
	err = ioutil.WriteFile(fullPath, []byte(contents), 0644)
	if err != nil {
		gc.log.Error("failed to write response to debug file", zap.String("filepath", fullPath), zap.Error(err))
		return
	}
	gc.log.Info("response written to debug file", zap.String("filepath", fullPath))
}
