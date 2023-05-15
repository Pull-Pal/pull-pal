package llm

import (
	"bytes"
	"strings"
	"text/template"
)

func (req DiffCommentRequest) String() string {
	return req.MustGetPrompt()
}

// MustGetPrompt only returns the prompt, but panics if the data in the request cannot populate the template.
func (req DiffCommentRequest) MustGetPrompt() string {
	prompt, err := req.GetPrompt()
	if err != nil {
		panic(err)
	}
	return prompt
}

// GetPrompt converts the information in the request to a prompt for an LLM.
func (req DiffCommentRequest) GetPrompt() (string, error) {
	tmpl, err := template.ParseFiles("./llm/prompts/comment-diff-request.tmpl")
	if err != nil {
		return "", err
	}

	replacer := strings.NewReplacer(
		"\\n", newlineLiteral,
	)
	req.File.Contents = replacer.Replace(req.File.Contents)

	var result bytes.Buffer
	err = tmpl.Execute(&result, req)
	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// String is a string representation of DiffCommentResponse.
func (res DiffCommentResponse) String() string {
	out := ""
	if res.Type == ResponseAnswer {
		out += "Type: Answer\n"
		out += res.Answer
		return out
	}

	out += "Type: Code Change\n"

	out += "Response:\n"
	out += res.Answer + "\n\n"
	out += "Files:\n"
	out += res.File.Path + ":\n```\n"
	out += res.File.Contents + "\n```\n"

	return out
}

func ParseDiffCommentResponse(llmResponse string) DiffCommentResponse {
	llmResponse = strings.TrimSpace(llmResponse)
	if llmResponse[0] == 'A' {
		answer := strings.TrimSpace(llmResponse[1:])
		return DiffCommentResponse{
			Type:   ResponseAnswer,
			Answer: answer,
		}
	}
	parts := strings.Split(llmResponse, "ppresponse:")

	filesSection := ""
	if len(parts) > 0 {
		filesSection = parts[0]
	}

	answer := ""
	if len(parts) > 1 {
		answer = strings.TrimSpace(parts[1])
	}

	files := parseFiles(filesSection)
	f := File{}
	if len(files) > 0 {
		f = files[0]
	}

	return DiffCommentResponse{
		Type:   ResponseCodeChange,
		Answer: answer,
		File:   f,
	}
}
