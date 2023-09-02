package llm

// File represents a file in a git repository.
type File struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
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
	Files []File `json:"files"`
	Notes string `json:"notes"`
}

// TODO support threads
type DiffCommentRequest struct {
	File     File
	Contents string
	Diff     string
}

type DiffCommentResponse struct {
	Type     ResponseType `json:"responseType"`
	Response string       `json:"response"`
	File     File         `json:"file"`
}
