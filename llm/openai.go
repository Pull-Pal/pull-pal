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
			// TODO make model configurable
			Model: openai.GPT4,
			//Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
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

	choice := resp.Choices[0].Message.Content

	// TODO make debug log when I figure out how to config that
	oc.log.Info("got response from llm", zap.String("output", choice))

	return ParseCodeChangeResponse(choice), nil
}
