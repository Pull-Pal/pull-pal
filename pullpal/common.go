package pullpal

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"strings"
	"time"

	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/vc"

	"go.uber.org/zap"
)

// IssueNotFound is returned when no issue can be found to generate a prompt for.
var IssueNotFound = errors.New("no issue found")

type Config struct {
	WaitDuration     time.Duration
	LocalRepoPath    string
	Repos            []string
	Self             vc.Author
	ListIssueOptions vc.ListIssueOptions
	Model            string
	OpenAIToken      string
}

// PullPal is the service responsible for:
//   - Interacting with git server (e.g. reading issues and making PRs on Github)
//   - Generating LLM prompts
//   - Parsing LLM responses
//   - Interacting with LLM (e.g. with GPT via OpenAI API)
type PullPal struct {
	ctx context.Context
	log *zap.Logger
	cfg Config

	repos        []pullPalRepo
	openAIClient *llm.OpenAIClient
}

type pullPalRepo struct {
	ctx context.Context
	log *zap.Logger

	listIssueOptions vc.ListIssueOptions
	ghClient         *vc.GithubClient
	localGitClient   *vc.LocalGitClient
	openAIClient     *llm.OpenAIClient
}

// NewPullPal creates a new "pull pal service", including setting up local version control and LLM integrations.
func NewPullPal(ctx context.Context, log *zap.Logger, cfg Config) (*PullPal, error) {
	openAIClient := llm.NewOpenAIClient(log.Named("openaiClient"), cfg.Model, cfg.OpenAIToken)

	ppRepos := []pullPalRepo{}
	for _, r := range cfg.Repos {
		fmt.Println(r)
		parts := strings.Split(r, "/")
		if len(parts) < 3 {
			continue
		}
		host := parts[0]
		owner := parts[1]
		name := parts[2]
		fmt.Println(host)
		fmt.Println(owner)
		fmt.Println(name)
		newRepo := vc.Repository{
			LocalPath:  filepath.Join(cfg.LocalRepoPath, owner, name),
			HostDomain: host,
			Name:       name,
			Owner: vc.Author{
				Handle: owner,
			},
		}
		ghClient, err := vc.NewGithubClient(ctx, log.Named("ghclient-"+r), cfg.Self, newRepo)
		if err != nil {
			return nil, err
		}
		localGitClient, err := vc.NewLocalGitClient(log.Named("gitclient-"+r), cfg.Self, newRepo)
		if err != nil {
			return nil, err
		}
		ppRepos = append(ppRepos, pullPalRepo{
			ctx: ctx,
			log: log,

			ghClient:       ghClient,
			localGitClient: localGitClient,
			openAIClient:   openAIClient,

			listIssueOptions: cfg.ListIssueOptions,
		})
	}
	if len(ppRepos) == 0 {
		return nil, errors.New("no repos set up")
	}

	return &PullPal{
		ctx: ctx,
		log: log,

		repos:        ppRepos,
		openAIClient: openAIClient,
		cfg:          cfg,
	}, nil
}

// Run starts pull pal as a fully automated service that periodically requests changes and creates pull requests based on them.
func (p *PullPal) Run() error {
	p.log.Info("Starting Pull Pal")
	// TODO gracefully handle context cancelation
	for {
		for _, r := range p.repos {
			err := r.checkIssuesAndComments()
			if err != nil {
				p.log.Error("issue checking repo for issues and comments", zap.Error(err))
			}
		}

		// TODO remove sleep
		p.log.Info("sleeping", zap.Duration("wait duration", p.cfg.WaitDuration))
		time.Sleep(p.cfg.WaitDuration)
	}
}

// checkIssuesAndComments will attempt to find and solve one issue and one comment, and then return.
func (p pullPalRepo) checkIssuesAndComments() error {
	p.log.Debug("checking github issues...")
	issues, err := p.ghClient.ListOpenIssues(p.listIssueOptions)
	if err != nil {
		p.log.Error("error listing issues", zap.Error(err))
		return err
	}

	if len(issues) == 0 {
		p.log.Debug("no issues found")
	} else {
		p.log.Info("picked issue to process")

		issue := issues[0]
		err = p.handleIssue(issue)
		if err != nil {
			// TODO leave comment if error (make configurable)
			p.log.Error("error handling issue", zap.Error(err))
			commentText := fmt.Sprintf("I ran into a problem working on this:\n```\n%s\n```", err.Error())
			err = p.ghClient.CommentOnIssue(issue.Number, commentText)
			if err != nil {
				p.log.Error("error commenting on issue with error", zap.Error(err))
				return err
			}
		}
	}

	p.log.Debug("checking pr comments...")
	comments, err := p.ghClient.ListOpenComments(vc.ListCommentOptions{
		Handles: p.listIssueOptions.Handles,
	})
	if err != nil {
		p.log.Error("error listing comments", zap.Error(err))
		return err
	}

	if len(comments) == 0 {
		p.log.Debug("no comments found")
	} else {
		p.log.Info("picked comment to process")

		comment := comments[0]
		err = p.handleComment(comment)
		if err != nil {
			// TODO leave comment if error (make configurable)
			p.log.Error("error handling comment", zap.Error(err))
			commentText := fmt.Sprintf("I ran into a problem working on this:\n```\n%s\n```", err.Error())
			err = p.ghClient.RespondToComment(comment.PRNumber, comment.ID, commentText)
			if err != nil {
				p.log.Error("error commenting on thread with error", zap.Error(err))
				return err
			}
		}
	}
	return nil
}

func (p *pullPalRepo) handleIssue(issue vc.Issue) error {
	err := p.ghClient.CommentOnIssue(issue.Number, "working on it")
	if err != nil {
		p.log.Error("error commenting on issue", zap.Error(err))
		return err
	}
	for _, label := range p.listIssueOptions.Labels {
		err = p.ghClient.RemoveLabelFromIssue(issue.Number, label)
		if err != nil {
			p.log.Error("error removing labels from issue", zap.Error(err))
			return err
		}
	}

	changeRequest, err := p.localGitClient.ParseIssueAndStartCommit(issue)
	if err != nil {
		p.log.Error("error parsing issue and starting commit", zap.Error(err))
		return err
	}

	changeResponse, err := p.openAIClient.EvaluateCCR(p.ctx, "", changeRequest)
	if err != nil {
		return err
	}

	randomNumber := rand.Intn(100) + 1
	newBranchName := fmt.Sprintf("fix-%d-%d", issue.Number, randomNumber)
	for _, f := range changeResponse.Files {
		p.log.Info("replacing or adding file", zap.String("path", f.Path), zap.String("contents", f.Contents))
		err = p.localGitClient.ReplaceOrAddLocalFile(f)
		if err != nil {
			return err
		}
	}

	commitMessage := fmt.Sprintf("%s\n\n%s\n\nResolves #%d", changeRequest.Subject, changeResponse.Notes, changeRequest.IssueNumber)
	p.log.Info("about to create commit", zap.String("message", commitMessage))
	err = p.localGitClient.FinishCommit(commitMessage)
	if err != nil {
		return err
	}

	p.log.Info("pushing to branch", zap.String("branchname", newBranchName))
	err = p.localGitClient.PushBranch(newBranchName)
	if err != nil {
		p.log.Info("error pushing to branch", zap.Error(err))
		return err
	}

	// open code change request
	// TODO don't hardcode main branch, make configurable
	_, url, err := p.ghClient.OpenCodeChangeRequest(changeRequest, changeResponse, newBranchName)
	if err != nil {
		return err
	}
	p.log.Info("successfully created PR", zap.String("URL", url))

	return nil
}

func (p *pullPalRepo) handleComment(comment vc.Comment) error {
	if comment.Branch == "" {
		return errors.New("no branch provided in comment")
	}

	file, err := p.localGitClient.GetLocalFile(comment.FilePath)
	if err != nil {
		return err
	}

	diffCommentRequest := llm.DiffCommentRequest{
		File:     file,
		Contents: comment.Body,
		Diff:     comment.DiffHunk,
	}
	p.log.Info("diff comment request", zap.String("req", diffCommentRequest.String()))

	diffCommentResponse, err := p.openAIClient.EvaluateDiffComment(p.ctx, "", diffCommentRequest)
	if err != nil {
		return err
	}

	if diffCommentResponse.Type == llm.ResponseCodeChange {
		p.log.Info("about to start commit")
		err = p.localGitClient.StartCommit()
		if err != nil {
			return err
		}
		p.log.Info("checking out branch", zap.String("name", comment.Branch))
		err = p.localGitClient.CheckoutRemoteBranch(comment.Branch)
		if err != nil {
			return err
		}
		p.log.Info("replacing or adding file", zap.String("path", diffCommentResponse.File.Path), zap.String("contents", diffCommentResponse.File.Contents))
		err = p.localGitClient.ReplaceOrAddLocalFile(diffCommentResponse.File)
		if err != nil {
			return err
		}

		commitMessage := "update based on comment"
		p.log.Info("about to create commit", zap.String("message", commitMessage))
		err = p.localGitClient.FinishCommit(commitMessage)
		if err != nil {
			return err
		}

		err = p.localGitClient.PushBranch(comment.Branch)
		if err != nil {
			return err
		}
	}

	err = p.ghClient.RespondToComment(comment.PRNumber, comment.ID, diffCommentResponse.Response)
	if err != nil {
		p.log.Error("error commenting on issue", zap.Error(err))
		return err
	}

	p.log.Info("responded addressed comment")

	return nil
}
