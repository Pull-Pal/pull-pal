package pullpal

import (
	"fmt"

	"github.com/mobyvb/pull-pal/llm"
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
	newBranchName := fmt.Sprintf("debug-branch")

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
