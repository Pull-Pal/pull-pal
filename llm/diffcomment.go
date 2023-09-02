package llm

import (
	"bytes"
	"encoding/json"
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
		out += res.Response
		return out
	}

	out += "Type: Code Change\n"

	out += "Response:\n"
	out += res.Response + "\n\n"
	out += "Files:\n"
	out += res.File.Path + ":\n```\n"
	out += res.File.Contents + "\n```\n"

	return out
}

func ParseDiffCommentResponse(llmResponse string) (DiffCommentResponse, error) {
	var response DiffCommentResponse
	err := json.Unmarshal([]byte(llmResponse), &response)
	return response, err
}
