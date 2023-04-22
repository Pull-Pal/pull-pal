package pullpal

import (
	"bytes"
	"strings"
	"text/template"
)

// PullPal is the service responsible for:
//  * Interacting with git server (e.g. reading issues and making PRs on Github)
//  * Generating LLM prompts
//  * Parsing LLM responses
//  * Interacting with LLM (e.g. with GPT via OpenAI API)
type PullPal struct {
}

// File represents a file in a git repository.
type File struct {
	Path     string
	Contents string
}

// CodeChangeRequest contains all necessary information for generating a prompt for a LLM.
type CodeChangeRequest struct {
	Files   []File
	Subject string
	Body    string
}

// String is the string representation of a CodeChangeRequest. Functionally, it contains the LLM prompt.
func (req CodeChangeRequest) String() string {
	prompt := req.MustGetPrompt()
	return "START OF PROMPT\n" + prompt + "\nEND OF PROMPT"
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
	tmpl, err := template.ParseFiles("../template/code-change-request.tmpl")
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

// CodeChangeResponse contains data derived from an LLM response to a prompt generated via a CodeChangeRequest.
type CodeChangeResponse struct {
	Files []File
	Notes string
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
func ParseCodeChangeResponse(llmResponse string) CodeChangeResponse {
	sections := strings.Split(llmResponse, "Notes:")

	filesSection := sections[0]
	notes := strings.TrimSpace(sections[1])

	files := parseFiles(filesSection)

	return CodeChangeResponse{
		Files: files,
		Notes: notes,
	}
}

// parseFiles process the "files" subsection of the LLM's response. It is a helper for GetCodeChangeResponse.
func parseFiles(filesSection string) []File {
	fileStringList := strings.Split(filesSection, "name:")
	// first item in the list is just gonna be "Files:"
	fileStringList = fileStringList[1:]

	fileList := make([]File, len(fileStringList))
	for i, f := range fileStringList {
		fileParts := strings.Split(f, "contents:")
		path := strings.TrimSpace(fileParts[0])
		contents := strings.TrimSpace(fileParts[1])
		contents = strings.Trim(contents, "```")

		fileList[i] = File{
			Path:     path,
			Contents: contents,
		}
	}

	return fileList
}
