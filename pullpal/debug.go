package pullpal

import (
	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/vc"
	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

func (p *PullPal) DebugGit() error {
	p.log.Info("Starting Pull Pal git debug")

	// create commit with file changes
	err := p.localGitClient.StartCommit()
	//err = p.ghClient.StartCommit()
	if err != nil {
		p.log.Error("error starting commit", zap.Error(err))
		return err
	}
	newBranchName := "debug-branch"

	for _, f := range []string{"a", "b"} {
		err = p.localGitClient.ReplaceOrAddLocalFile(llm.File{
			Path:     f,
			Contents: "hello",
		})
		if err != nil {
			p.log.Error("error replacing or adding file", zap.Error(err))
			return err
		}
	}

	commitMessage := "debug commit message"
	err = p.localGitClient.FinishCommit(commitMessage)
	if err != nil {
		p.log.Error("error finishing commit", zap.Error(err))
		return err
	}

	err = p.localGitClient.PushBranch(newBranchName)
	if err != nil {
		p.log.Error("error pushing branch", zap.Error(err))
		return err
	}

	return nil
}

// todo dont require args for listing comments
func (p *PullPal) DebugGithub(handles []string) error {
	p.log.Info("Starting Pull Pal Github debug")

	issues, err := p.ghClient.ListOpenIssues(p.listIssueOptions)
	if err != nil {
		p.log.Error("error listing issues", zap.Error(err))
		return err
	}
	for _, i := range issues {
		p.log.Info("got issue", zap.String("issue", i.String()))
	}

	comments, err := p.ghClient.ListOpenComments(vc.ListCommentOptions{
		Handles: handles,
	})
	if err != nil {
		p.log.Error("error listing comments", zap.Error(err))
		return err
	}
	for _, c := range comments {
		p.log.Info("got comment", zap.String("comment", c.String()))
	}

	return nil
}

func (p *PullPal) DebugLLM() error {
	p.log.Info("Starting Pull Pal llm debug")

	file := llm.File{
		Path:     "main.go",
		Contents: `package main\n\nimport (\n    "net/http"\n)\n\nfunc main() {\n    fs := http.FileServer(http.Dir("static"))\n    http.Handle("/", fs)\n\n    http.ListenAndServe(":7777", nil)\n}\n\n\n  \n\n   \n  `,
	}

	codeChangeRequest := llm.CodeChangeRequest{
		Files:       []llm.File{file},
		Subject:     "update port and add endpoint",
		Body:        "use port 8080 for the server in main.go. Also add an endpoint at GET /api/numbers that returns a random integer between 2 and 10",
		IssueNumber: 1234,
	}

	p.log.Info("CODE CHANGE REQUEST", zap.String("request", codeChangeRequest.String()))

	diffCommentRequestChange := llm.DiffCommentRequest{
		File:     file,
		Contents: "remove this unnecessary whitespace at the end",
		Diff:     "@@ -0,0 +1,15 @@\n+package main\n+    \n+    import (\n+        \"net/http\"\n+    )\n+    \n+    func main() {\n+        fs := http.FileServer(http.Dir(\"static\"))\n+        http.Handle(\"/\", fs)\n+    \n+        http.ListenAndServe(\":7777\", nil)\n+    }\n+",
	}
	p.log.Info("DIFF COMMENT REQUEST CODECHANGE", zap.String("request", diffCommentRequestChange.String()))

	diffCommentRequestQuestion := llm.DiffCommentRequest{
		File:     file,
		Contents: "what does this Handle line do?",
		Diff:     "@@ -0,0 +1,15 @@\n+package main\n+    \n+    import (\n+        \"net/http\"\n+    )\n+    \n+    func main() {\n+        fs := http.FileServer(http.Dir(\"static\"))\n+        http.Handle(\"/\", fs)\n",
	}
	p.log.Info("DIFF COMMENT REQUEST QUESTION", zap.String("request", diffCommentRequestQuestion.String()))

	for _, m := range []string{openai.GPT3Dot5Turbo, openai.GPT4} {
		p.log.Info("testing with openai api", zap.String("MODEL", m))

		p.log.Info("testing code change request")
		res, err := p.openAIClient.EvaluateCCR(p.ctx, m, codeChangeRequest)
		if err != nil {
			p.log.Error("error evaluating code change request for model", zap.Error(err))
			continue
		}
		p.log.Info("openai api response", zap.String("model", m), zap.String("response", res.String()))

		p.log.Info("testing diff comment code change request")
		diffRes, err := p.openAIClient.EvaluateDiffComment(p.ctx, m, diffCommentRequestChange)
		if err != nil {
			p.log.Error("error evaluating diff comment request for model", zap.Error(err))
			continue
		}
		p.log.Info("openai api response", zap.String("model", m), zap.String("response", diffRes.String()))

		p.log.Info("testing diff comment question request")
		diffRes, err = p.openAIClient.EvaluateDiffComment(p.ctx, m, diffCommentRequestQuestion)
		if err != nil {
			p.log.Error("error evaluating diff comment request for model", zap.Error(err))
			continue
		}
		p.log.Info("openai api response", zap.String("model", m), zap.String("response", diffRes.String()))

	}

	// TODO group errors  and return
	return nil
}
