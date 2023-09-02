package llm

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/sashabaranov/go-openai"
	"go.uber.org/zap"
)

type OpenAIClient struct {
	log          *zap.Logger
	client       *openai.Client
	debugDir     string
	defaultModel string
}

func NewOpenAIClient(log *zap.Logger, defaultModel, token, debugDir string) *OpenAIClient {
	return &OpenAIClient{
		log:          log,
		client:       openai.NewClient(token),
		defaultModel: defaultModel,
		debugDir:     debugDir,
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

	oc.log.Info("got response from llm")
	if oc.debugDir != "" {
		subdir := path.Join(oc.debugDir, "codechangeresponse")
		err = os.MkdirAll(subdir, os.ModePerm)
		if err != nil {
			oc.log.Error("failed to ensure debug directory existed", zap.String("filepath", subdir), zap.Error(err))
		} else {
			fullPath := path.Join(subdir, fmt.Sprintf("%d-%d.json", req.IssueNumber, time.Now().Unix()))
			err = ioutil.WriteFile(fullPath, []byte(choice), 0644)
			if err != nil {
				oc.log.Error("failed to write response to debug file", zap.String("filepath", fullPath), zap.Error(err))
			} else {
				oc.log.Info("response written to debug file", zap.String("filepath", fullPath))
			}
		}
	}

	return ParseCodeChangeResponse(choice)
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

	oc.log.Info("got response from llm", zap.String("output", choice))
	// TODO
	if oc.debugDir != "" {
		subdir := path.Join(oc.debugDir, "diffcommentresponse")
		err = os.MkdirAll(subdir, os.ModePerm)
		if err != nil {
			oc.log.Error("failed to ensure debug directory existed", zap.String("filepath", subdir), zap.Error(err))
		} else {
			fullPath := path.Join(subdir, fmt.Sprintf("%d-%d.json", req.PRNumber, time.Now().Unix()))
			err = ioutil.WriteFile(fullPath, []byte(choice), 0644)
			if err != nil {
				oc.log.Error("failed to write response to debug file", zap.String("filepath", fullPath), zap.Error(err))
			} else {
				oc.log.Info("response written to debug file", zap.String("filepath", fullPath))
			}
		}
	}

	return ParseDiffCommentResponse(choice)
}
