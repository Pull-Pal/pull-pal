package llm

import (
	"context"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type OpenAIClient struct {
	log          *zap.Logger
	client       *openai.Client
	defaultModel string
}

func NewOpenAIClient(log *zap.Logger, defaultModel, token string) *OpenAIClient {
	return &OpenAIClient{
		log:          log,
		client:       openai.NewClient(token),
		defaultModel: defaultModel,
	}
}

func (oc *OpenAIClient) EvaluateCCR(ctx context.Context, model string, req CodeChangeRequest) (res CodeChangeResponse, err error) {
	if model == "" {
		model = oc.defaultModel
	}
	resp, err := oc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
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

func (oc *OpenAIClient) EvaluateDiffComment(ctx context.Context, model string, req DiffCommentRequest) (res DiffCommentResponse, err error) {
	if model == "" {
		model = oc.defaultModel
	}
	resp, err := oc.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: model,
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

	return ParseDiffCommentResponse(choice), nil
}
