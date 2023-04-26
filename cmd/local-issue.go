package cmd

import (
	"fmt"

	"github.com/mobyvb/pull-pal/vc"

	"github.com/spf13/cobra"
)

var localIssueCmd = &cobra.Command{
	Use:   "local-issue",
	Short: "Processes a locally-defined/provided issue rather than remotely reading one from the Github repo",
	// TODO csv filepath as arg?
	// Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		newIssue := vc.Issue{
			Subject: "a few updates",
			Body:    "Add a quote from Frodo to the README.md and index.html files.\nSwitch main.go to port 7777.\nFiles:index.html,README.md,main.go",
			Author: vc.Author{
				Handle: "mobyvb",
			},
		}
		err = p.MakeLocalChange(newIssue)
		if err != nil {
			fmt.Println("err making local change", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(localIssueCmd)
}
