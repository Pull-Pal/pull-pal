package cmd

import (
	"fmt"

	"github.com/mobyvb/pull-pal/pullpal"
	"github.com/mobyvb/pull-pal/vc"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var listIssuesCmd = &cobra.Command{
	Use:   "list-issues",
	Short: "Lists github issues meeting the configured criteria",
	Long:  "Lists github issues meeting the configured criteria",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()
		fmt.Println("list issues called")

		log := zap.L()

		author := vc.Author{
			Email:  cfg.selfEmail,
			Handle: cfg.selfHandle,
			Token:  cfg.githubToken,
		}
		repo := vc.Repository{
			LocalPath:  cfg.localRepoPath,
			HostDomain: cfg.repoDomain,
			Name:       cfg.repoName,
			Owner: vc.Author{
				Handle: cfg.repoHandle,
			},
		}
		p, err := pullpal.NewPullPal(cmd.Context(), log.Named("pullpal"), author, repo)
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
