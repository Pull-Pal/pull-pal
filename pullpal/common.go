package pullpal

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/vc"

	"github.com/atotto/clipboard"
	"go.uber.org/zap"
)

// IssueNotFound is returned when no issue can be found to generate a prompt for.
var IssueNotFound = errors.New("no issue found")

// PullPal is the service responsible for:
//  * Interacting with git server (e.g. reading issues and making PRs on Github)
//  * Generating LLM prompts
//  * Parsing LLM responses
//  * Interacting with LLM (e.g. with GPT via OpenAI API)
type PullPal struct {
	ctx              context.Context
	log              *zap.Logger
	listIssueOptions vc.ListIssueOptions

	vcClient       vc.VCClient
	localGitClient *vc.LocalGitClient
	openAIClient   *llm.OpenAIClient
}

// NewPullPal creates a new "pull pal service", including setting up local version control and LLM integrations.
func NewPullPal(ctx context.Context, log *zap.Logger, listIssueOptions vc.ListIssueOptions, self vc.Author, repo vc.Repository, openAIToken string) (*PullPal, error) {
	ghClient, err := vc.NewGithubClient(ctx, log, self, repo)
	if err != nil {
		return nil, err
	}
	localGitClient, err := vc.NewLocalGitClient(self, repo)
	if err != nil {
		return nil, err
	}

	return &PullPal{
		ctx:              ctx,
		log:              log,
		listIssueOptions: listIssueOptions,

		vcClient:       ghClient,
		localGitClient: localGitClient,
		openAIClient:   llm.NewOpenAIClient(log.Named("openaiClient"), openAIToken),
	}, nil
}

// Run starts pull pal as a fully automated service that periodically requests changes and creates pull requests based on them.
func (p *PullPal) Run() error {
	p.log.Info("Starting Pull Pal")
	// TODO gracefully handle context cancelation
	for {
		issues, err := p.vcClient.ListOpenIssues(p.listIssueOptions)
		if err != nil {
			p.log.Error("error listing issues", zap.Error(err))
			continue
		}

		if len(issues) == 0 {
			// todo don't sleep
			p.log.Info("no issues found. sleeping for 5 mins")
			time.Sleep(5 * time.Minute)
			continue
		}

		issue := issues[0]

		// remove file list from issue body
		// TODO do this better and probably somewhere else
		parts := strings.Split(issue.Body, "Files:")
		issue.Body = parts[0]

		fileList := []string{}
		if len(parts) > 1 {
			fileList = strings.Split(parts[1], ",")
		}

		// get file contents from local git repository
		files := []llm.File{}
		for _, path := range fileList {
			path = strings.TrimSpace(path)
			nextFile, err := p.vcClient.GetLocalFile(path)
			if err != nil {
				p.log.Error("error getting file from vcclient", zap.Error(err))
				continue
			}
			files = append(files, nextFile)
		}

		changeRequest := llm.CodeChangeRequest{
			Subject: issue.Subject,
			Body:    issue.Body,
			IssueID: issue.ID,
			Files:   files,
		}

		changeResponse, err := p.openAIClient.EvaluateCCR(p.ctx, changeRequest)
		if err != nil {
			p.log.Error("error getting response from openai", zap.Error(err))
			continue

		}

		// parse llm response
		//codeChangeResponse := llm.ParseCodeChangeResponse(llmResponse)

		// create commit with file changes
		err = p.vcClient.StartCommit()
		if err != nil {
			p.log.Error("error starting commit", zap.Error(err))
			continue
		}
		for _, f := range changeResponse.Files {
			err = p.vcClient.ReplaceOrAddLocalFile(f)
			if err != nil {
				p.log.Error("error replacing or adding file", zap.Error(err))
				continue
			}
		}

		commitMessage := changeRequest.Subject + "\n\n" + changeResponse.Notes + "\n\nResolves: #" + changeRequest.IssueID
		err = p.vcClient.FinishCommit(commitMessage)
		if err != nil {
			p.log.Error("error finshing commit", zap.Error(err))
			continue
		}

		// open code change request
		_, url, err := p.vcClient.OpenCodeChangeRequest(changeRequest, changeResponse)
		if err != nil {
			p.log.Error("error opening PR", zap.Error(err))
		}
		p.log.Info("successfully created PR", zap.String("URL", url))

		p.log.Info("going to sleep for five mins")
		time.Sleep(5 * time.Minute)
	}

	return nil
}

// PickIssueToFile is the same as PickIssue, but the changeRequest is converted to a string and written to a file.
func (p *PullPal) PickIssueToFile(promptPath string) (issue vc.Issue, changeRequest llm.CodeChangeRequest, err error) {
	issue, changeRequest, err = p.PickIssue()
	if err != nil {
		return issue, changeRequest, err
	}

	prompt, err := changeRequest.GetPrompt()
	if err != nil {
		return issue, changeRequest, err
	}

	err = ioutil.WriteFile(promptPath, []byte(prompt), 0644)
	return issue, changeRequest, err
}

// PickIssueToClipboard is the same as PickIssue, but the changeRequest is converted to a string and copied to the clipboard.
func (p *PullPal) PickIssueToClipboard() (issue vc.Issue, changeRequest llm.CodeChangeRequest, err error) {
	issue, changeRequest, err = p.PickIssue()
	if err != nil {
		return issue, changeRequest, err
	}

	prompt, err := changeRequest.GetPrompt()
	if err != nil {
		return issue, changeRequest, err
	}

	err = clipboard.WriteAll(prompt)
	return issue, changeRequest, err
}

// PickIssue selects an issue from the version control server and returns the selected issue, as well as the LLM prompt needed to fulfill the request.
func (p *PullPal) PickIssue() (issue vc.Issue, changeRequest llm.CodeChangeRequest, err error) {
	// TODO I should be able to pass in settings for listing issues from here
	issues, err := p.vcClient.ListOpenIssues(p.listIssueOptions)
	if err != nil {
		return issue, changeRequest, err
	}

	if len(issues) == 0 {
		return issue, changeRequest, IssueNotFound
	}

	issue = issues[0]

	// remove file list from issue body
	// TODO do this better
	parts := strings.Split(issue.Body, "Files:")
	issue.Body = parts[0]

	fileList := []string{}
	if len(parts) > 1 {
		fileList = strings.Split(parts[1], ",")
	}

	// get file contents from local git repository
	files := []llm.File{}
	for _, path := range fileList {
		path = strings.TrimSpace(path)
		nextFile, err := p.vcClient.GetLocalFile(path)
		if err != nil {
			return issue, changeRequest, err
		}
		files = append(files, nextFile)
	}

	changeRequest.Subject = issue.Subject
	changeRequest.Body = issue.Body
	changeRequest.IssueID = issue.ID
	changeRequest.Files = files

	return issue, changeRequest, nil
}

// ProcessResponseFromFile is the same as ProcessResponse, but the response is inputted into a file rather than passed directly as an argument.
func (p *PullPal) ProcessResponseFromFile(codeChangeRequest llm.CodeChangeRequest, llmResponsePath string) (url string, err error) {
	data, err := ioutil.ReadFile(llmResponsePath)
	if err != nil {
		return "", err
	}
	return p.ProcessResponse(codeChangeRequest, string(data))
}

// ProcessResponse parses the llm response, updates files in the local git repo accordingly, and opens a new code change request (e.g. Github PR).
func (p *PullPal) ProcessResponse(codeChangeRequest llm.CodeChangeRequest, llmResponse string) (url string, err error) {
	// 1. parse llm response
	codeChangeResponse := llm.ParseCodeChangeResponse(llmResponse)

	// 2. create commit with file changes
	err = p.vcClient.StartCommit()
	if err != nil {
		return "", err
	}
	for _, f := range codeChangeResponse.Files {
		err = p.vcClient.ReplaceOrAddLocalFile(f)
		if err != nil {
			return "", err
		}
	}

	commitMessage := codeChangeRequest.Subject + "\n\n" + codeChangeResponse.Notes + "\n\nResolves: #" + codeChangeRequest.IssueID
	err = p.vcClient.FinishCommit(commitMessage)
	if err != nil {
		return "", err
	}

	// 3. open code change request
	_, url, err = p.vcClient.OpenCodeChangeRequest(codeChangeRequest, codeChangeResponse)
	return url, err
}

// ListIssues gets a list of all issues meeting the provided criteria.
func (p *PullPal) ListIssues(handles, labels []string) ([]vc.Issue, error) {
	issues, err := p.vcClient.ListOpenIssues(vc.ListIssueOptions{
		Handles: handles,
		Labels:  labels,
	})
	if err != nil {
		return nil, err
	}

	return issues, nil
}

// ListComments gets a list of all comments meeting the provided criteria on a PR.
func (p *PullPal) ListComments(changeID string, handles []string) ([]vc.Comment, error) {
	comments, err := p.vcClient.ListOpenComments(vc.ListCommentOptions{
		ChangeID: changeID,
		Handles:  handles,
	})
	if err != nil {
		return nil, err
	}

	return comments, nil
}

func (p *PullPal) MakeLocalChange(issue vc.Issue) error {
	// remove file list from issue body
	// TODO do this better
	parts := strings.Split(issue.Body, "Files:")
	issue.Body = parts[0]

	fileList := []string{}
	if len(parts) > 1 {
		fileList = strings.Split(parts[1], ",")
	}

	// get file contents from local git repository
	files := []llm.File{}
	for _, path := range fileList {
		path = strings.TrimSpace(path)
		nextFile, err := p.vcClient.GetLocalFile(path)
		if err != nil {
			return err
		}
		files = append(files, nextFile)
	}

	changeRequest := llm.CodeChangeRequest{
		Subject: issue.Subject,
		Body:    issue.Body,
		IssueID: issue.ID,
		Files:   files,
	}

	res, err := p.openAIClient.EvaluateCCR(p.ctx, changeRequest)
	if err != nil {
		return err
	}

	fmt.Println("response from openai")
	fmt.Println(res)

	return nil
}
