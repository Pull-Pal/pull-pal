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

	debugFilePrefix := fmt.Sprintf("%d-%d", req.IssueNumber, time.Now().Unix())
	oc.writeDebug("codechangeresponse", debugFilePrefix+"-req.txt", req.String())
	oc.writeDebug("codechangeresponse", debugFilePrefix+"-res.yaml", choice)

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

	debugFilePrefix := fmt.Sprintf("%d-%d", req.PRNumber, time.Now().Unix())
	oc.writeDebug("diffcommentresponse", debugFilePrefix+"-req.txt", req.String())
	oc.writeDebug("diffcommentresponse", debugFilePrefix+"-res.yaml", choice)

	return ParseDiffCommentResponse(choice)
}

func (oc *OpenAIClient) writeDebug(subdir, filename, contents string) {
	if oc.debugDir == "" {
		return
	}

	fullFolderPath := path.Join(oc.debugDir, subdir)

	err := os.MkdirAll(fullFolderPath, os.ModePerm)
	if err != nil {
		oc.log.Error("failed to ensure debug directory existed", zap.String("folderpath", fullFolderPath), zap.Error(err))
		return
	}

	fullPath := path.Join(fullFolderPath, filename)
	err = ioutil.WriteFile(fullPath, []byte(contents), 0644)
	if err != nil {
		oc.log.Error("failed to write response to debug file", zap.String("filepath", fullPath), zap.Error(err))
		return
	}
	oc.log.Info("response written to debug file", zap.String("filepath", fullPath))
}
