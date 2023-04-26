package llm

import (
	"context"
	"fmt"

	"github.com/sashabaranov/go-openai"
)

type OpenAIClient struct {
	client *openai.Client
}

func NewOpenAIClient(token string) *OpenAIClient {
	return &OpenAIClient{
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
		fmt.Printf("ChatCompletion error: %v\n", err)
		return res, err
	}

	// TODO use different choices/different options in different branches/worktrees?
	choice := resp.Choices[0].Message.Content

	return ParseCodeChangeResponse(choice), nil
}
