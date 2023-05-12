package llm

import (
	"strings"
)

// File represents a file in a git repository.
type File struct {
	Path     string
	Contents string
}

type ResponseType int

const (
	ResponseAnswer ResponseType = iota
	ResponseCodeChange
)

// CodeChangeRequest contains all necessary information for generating a prompt for a LLM.
type CodeChangeRequest struct {
	Files       []File
	Subject     string
	Body        string
	IssueNumber int
	BaseBranch  string
}

// CodeChangeResponse contains data derived from an LLM response to a prompt generated via a CodeChangeRequest.
type CodeChangeResponse struct {
	Files []File
	Notes string
}

// TODO support threads
type DiffCommentRequest struct {
	File     File
	Contents string
	Diff     string
}

type DiffCommentResponse struct {
	Type   ResponseType
	Answer string
	File   File
}

// parseFiles process the "files" subsection of the LLM's response. It is a helper for GetCodeChangeResponse.
func parseFiles(filesSection string) []File {
	fileStringList := strings.Split(filesSection, "ppname:")
	if len(fileStringList) < 2 {
		return []File{}
	}
	// first item in the list is just gonna be "Files:"
	fileStringList = fileStringList[1:]

	replacer := strings.NewReplacer(
		"\\n", "\n",
		"\\\"", "\"",
		"```", "",
	)
	fileList := make([]File, len(fileStringList))
	for i, f := range fileStringList {
		fileParts := strings.Split(f, "ppcontents:")
		if len(fileParts) < 2 {
			continue
		}
		path := replacer.Replace(fileParts[0])
		path = strings.TrimSpace(path)

		contents := replacer.Replace(fileParts[1])
		contents = strings.TrimSpace(contents)

		fileList[i] = File{
			Path:     path,
			Contents: contents,
		}
	}

	return fileList
}
