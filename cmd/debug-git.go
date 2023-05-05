package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var debugGitCmd = &cobra.Command{
	Use:   "debug-git",
	Short: "debug git functionality",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		err = p.DebugGit()
		if err != nil {
			fmt.Println("err debugging git", err)
			return
		}
	},
}

var debugGithubCmd = &cobra.Command{
	Use:   "debug-github",
	Short: "debug github functionality",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		err = p.DebugGithub(cfg.usersToListenTo)
		if err != nil {
			fmt.Println("err debugging github", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(debugGitCmd)
	rootCmd.AddCommand(debugGithubCmd)
}
