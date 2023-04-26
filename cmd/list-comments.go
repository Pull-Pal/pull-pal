package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var listCommentsCmd = &cobra.Command{
	Use:   "list-comments",
	Short: "Lists comments on a Github PR meeting the configured criteria",
	Long:  "Lists comments on a Github PR meeting the configured criteria",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		prID := args[0]
		issueList, err := p.ListComments(prID, cfg.usersToListenTo)
		if err != nil {
			fmt.Println("error listing issues", err)
			return
		}
		fmt.Println(issueList)
	},
}

func init() {
	rootCmd.AddCommand(listCommentsCmd)
}
