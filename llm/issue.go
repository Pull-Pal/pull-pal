package llm

import (
	"bytes"
	"encoding/json"
	"text/template"
)

// String is the string representation of a CodeChangeRequest. Functionally, it contains the LLM prompt.
func (req CodeChangeRequest) String() string {
	return req.MustGetPrompt()
}

// MustGetPrompt only returns the prompt, but panics if the data in the request cannot populate the template.
func (req CodeChangeRequest) MustGetPrompt() string {
	prompt, err := req.GetPrompt()
	if err != nil {
		panic(err)
	}
	return prompt
}

// GetPrompt converts the information in the request to a prompt for an LLM.
func (req CodeChangeRequest) GetPrompt() (string, error) {
	tmpl, err := template.ParseFiles("./llm/prompts/code-change-request.tmpl")
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

// String is a string representation of CodeChangeResponse.
func (res CodeChangeResponse) String() string {
	out := "Notes:\n"
	out += res.Notes + "\n\n"
	out += "Files:\n"
	for _, f := range res.Files {
		out += f.Path + ":\n```\n"
		out += f.Contents + "\n```\n"
	}

	return out
}

// ParseCodeChangeResponse parses the LLM's response to CodeChangeRequest (string) into a CodeChangeResponse.
func ParseCodeChangeResponse(llmResponse string) (CodeChangeResponse, error) {
	var response CodeChangeResponse
	err := json.Unmarshal([]byte(llmResponse), &response)
	return response, err
}
