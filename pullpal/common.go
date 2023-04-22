package pullpal

import (
	"context"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/vc"

	"go.uber.org/zap"
)

// PullPal is the service responsible for:
//  * Interacting with git server (e.g. reading issues and making PRs on Github)
//  * Generating LLM prompts
//  * Parsing LLM responses
//  * Interacting with LLM (e.g. with GPT via OpenAI API)
type PullPal struct {
	ctx context.Context
	log *zap.Logger

	vcClient vc.VCClient
}

// NewPullPal creates a new "pull pal service", including setting up local version control and LLM integrations.
func NewPullPal(ctx context.Context, log *zap.Logger, self vc.Author, repo vc.Repository) (*PullPal, error) {
	ghClient, err := vc.NewGithubClient(ctx, log, self, repo)
	if err != nil {
		log.Error("Failed to setup Github client.", zap.Error(err))
	}

	return &PullPal{
		ctx: ctx,
		log: log,

		vcClient: ghClient,
	}, nil
}

// IssueNotFound is returned when no issue can be found to generate a prompt for.
var IssueNotFound = errors.New("no issue found")

// PickIssue selects an issue from the version control server and returns the selected issue, as well as the LLM prompt needed to fulfill the request.
func (p *PullPal) PickIssue() (issue vc.Issue, changeRequest llm.CodeChangeRequest, err error) {
	// TODO I should be able to pass in settings for listing issues from here
	issues, err := p.vcClient.ListOpenIssues()
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
