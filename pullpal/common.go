package pullpal

import (
	"context"

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
