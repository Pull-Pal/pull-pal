package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listIssuesCmd = &cobra.Command{
	Use:   "list-issues",
	Short: "Lists github issues meeting the configured criteria",
	Long:  "Lists github issues meeting the configured criteria",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		issueList, err := p.ListIssues(cfg.usersToListenTo, cfg.requiredIssueLabels)
		if err != nil {
			fmt.Println("error listing issues", err)
			return
		}
		fmt.Println(issueList)
	},
}

func init() {
	rootCmd.AddCommand(listIssuesCmd)
}
