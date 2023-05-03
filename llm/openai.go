package llm

import (
	"context"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type OpenAIClient struct {
	log    *zap.Logger
	client *openai.Client
}

func NewOpenAIClient(log *zap.Logger, token string) *OpenAIClient {
	return &OpenAIClient{
		log:    log,
		client: openai.NewClient(token),
	}
}

func (oc *OpenAIClient) EvaluateCCR(ctx context.Context, req CodeChangeRequest) (res CodeChangeResponse, err error) {
	resp, err := oc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					// TODO is this the correct role for my prompts?
					Role:    openai.ChatMessageRoleUser,
					Content: req.String(),
				},
			},
		},
	)
	if err != nil {
		oc.log.Error("chat completion error", zap.Error(err))
		return res, err
	}

	// TODO use different choices/different options in different branches/worktrees?
	choice := resp.Choices[0].Message.Content

	oc.log.Debug("got response from llm", zap.String("output", choice))

	return ParseCodeChangeResponse(choice), nil
}
