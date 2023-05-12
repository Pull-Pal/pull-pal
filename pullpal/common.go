package pullpal

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/vc"

	"go.uber.org/zap"
)

// IssueNotFound is returned when no issue can be found to generate a prompt for.
var IssueNotFound = errors.New("no issue found")

// PullPal is the service responsible for:
//   - Interacting with git server (e.g. reading issues and making PRs on Github)
//   - Generating LLM prompts
//   - Parsing LLM responses
//   - Interacting with LLM (e.g. with GPT via OpenAI API)
type PullPal struct {
	ctx              context.Context
	log              *zap.Logger
	listIssueOptions vc.ListIssueOptions

	ghClient       *vc.GithubClient
	localGitClient *vc.LocalGitClient
	openAIClient   *llm.OpenAIClient
}

// NewPullPal creates a new "pull pal service", including setting up local version control and LLM integrations.
func NewPullPal(ctx context.Context, log *zap.Logger, listIssueOptions vc.ListIssueOptions, self vc.Author, repo vc.Repository, model string, openAIToken string) (*PullPal, error) {
	ghClient, err := vc.NewGithubClient(ctx, log, self, repo)
	if err != nil {
		return nil, err
	}
	localGitClient, err := vc.NewLocalGitClient(log, self, repo)
	if err != nil {
		return nil, err
	}

	return &PullPal{
		ctx:              ctx,
		log:              log,
		listIssueOptions: listIssueOptions,

		ghClient:       ghClient,
		localGitClient: localGitClient,
		openAIClient:   llm.NewOpenAIClient(log.Named("openaiClient"), model, openAIToken),
	}, nil
}

// Run starts pull pal as a fully automated service that periodically requests changes and creates pull requests based on them.
func (p *PullPal) Run() error {
	p.log.Info("Starting Pull Pal")
	// TODO gracefully handle context cancelation
	for {
		p.log.Info("checking github issues...")
		issues, err := p.ghClient.ListOpenIssues(p.listIssueOptions)
		if err != nil {
			p.log.Error("error listing issues", zap.Error(err))
			return err
		}

		if len(issues) == 0 {
			p.log.Info("no issues found")
		} else {
			p.log.Info("picked issue to process")

			issue := issues[0]
			err = p.handleIssue(issue)
			if err != nil {
				// TODO leave comment if error (make configurable)
				p.log.Error("error handling issue", zap.Error(err))
			}
		}

		p.log.Info("checking pr comments...")
		comments, err := p.ghClient.ListOpenComments(vc.ListCommentOptions{
			Handles: p.listIssueOptions.Handles,
		})
		if err != nil {
			p.log.Error("error listing comments", zap.Error(err))
			return err
		}

		if len(comments) == 0 {
			p.log.Info("no comments found")
		} else {
			p.log.Info("picked comment to process")

			comment := comments[0]
			err = p.handleComment(comment)
			if err != nil {
				// TODO leave comment if error (make configurable)
				p.log.Error("error handling comment", zap.Error(err))
			}
		}

		// TODO remove sleep
		p.log.Info("sleeping 30s")
		time.Sleep(30 * time.Second)
	}
}

func (p *PullPal) handleIssue(issue vc.Issue) error {
	issueNumber, err := strconv.Atoi(issue.ID)
	if err != nil {
		p.log.Error("error converting issue ID to int", zap.Error(err))
		return err
	}

	err = p.ghClient.CommentOnIssue(issueNumber, "working on it")
	if err != nil {
		p.log.Error("error commenting on issue", zap.Error(err))
		return err
	}
	for _, label := range p.listIssueOptions.Labels {
		err = p.ghClient.RemoveLabelFromIssue(issueNumber, label)
		if err != nil {
			p.log.Error("error removing labels from issue", zap.Error(err))
			return err
		}
	}

	changeRequest, err := p.localGitClient.ParseIssueAndStartCommit(issue)
	if err != nil {
		return err
	}

	changeResponse, err := p.openAIClient.EvaluateCCR(p.ctx, "", changeRequest)
	if err != nil {
		return err
	}

	newBranchName := fmt.Sprintf("fix-%s", changeRequest.IssueID)
	for _, f := range changeResponse.Files {
		p.log.Info("replacing or adding file", zap.String("path", f.Path), zap.String("contents", f.Contents))
		err = p.localGitClient.ReplaceOrAddLocalFile(f)
		if err != nil {
			return err
		}
	}

	commitMessage := changeRequest.Subject + "\n\n" + changeResponse.Notes + "\n\nResolves: #" + changeRequest.IssueID
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

func (p *PullPal) handleComment(comment vc.Comment) error {
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

	err = p.ghClient.RespondToComment(comment.PRNumber, comment.ID, diffCommentResponse.Answer)
	if err != nil {
		p.log.Error("error commenting on issue", zap.Error(err))
		return err
	}

	p.log.Info("responded addressed comment")

	return nil
}
