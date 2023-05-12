package vc_test

import (
	"testing"

	"github.com/mobyvb/pull-pal/vc"

	"github.com/stretchr/testify/require"
)

func TestParseIssueBody(t *testing.T) {
	var testCases = []struct {
		testcase string
		body     string
		parsed   vc.IssueBody
	}{
		{
			"simple issue",
			`
add an html file
			`,
			vc.IssueBody{
				PromptBody: "add an html file",
				BaseBranch: "main",
			},
		},
		{
			"issue with explicit file list",
			`
add an html file
and also a go file
read a readme file too

---

FiLeS: index.html, README.md ,main.go   
			`,
			vc.IssueBody{
				PromptBody: "add an html file\nand also a go file\nread a readme file too",
				BaseBranch: "main",
				FilePaths:  []string{"index.html", "README.md", "main.go"},
			},
		},
		{
			"issue with a custom base branch",
			`
add an html file
---
base:  some-base-branch 
			`,
			vc.IssueBody{
				PromptBody: "add an html file",
				BaseBranch: "some-base-branch",
			},
		},
		{
			"issue with an explicit base branch and file list",
			`
add an html file
---
base:  some-base-branch 
files: index.html, main.go
			`,
			vc.IssueBody{
				PromptBody: "add an html file",
				BaseBranch: "some-base-branch",
				FilePaths:  []string{"index.html", "main.go"},
			},
		},
		{
			"issue with garbage in config section",
			`
add an html file
---
asdf:
files: index.html, main.go
: asdfsadf
base:  some-base-branch 
asdfjljldsfj
nonexistentoption: asdf
			`,
			vc.IssueBody{
				PromptBody: "add an html file",
				BaseBranch: "some-base-branch",
				FilePaths:  []string{"index.html", "main.go"},
			},
		},
	}
	for _, tt := range testCases {
		t.Log("testing case:", tt.testcase)
		parsed := vc.ParseIssueBody(tt.body)
		require.Equal(t, tt.parsed.PromptBody, parsed.PromptBody)
		require.Equal(t, tt.parsed.BaseBranch, parsed.BaseBranch)
		require.Equal(t, len(tt.parsed.FilePaths), len(parsed.FilePaths))
		for i, p := range tt.parsed.FilePaths {
			require.Equal(t, p, parsed.FilePaths[i])
		}
	}
}
