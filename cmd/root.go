package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mobyvb/pull-pal/llm"
	"github.com/mobyvb/pull-pal/pullpal"
	"github.com/mobyvb/pull-pal/vc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "pull-pal",
	Short: "A bot that uses large language models to act as a collaborator on a git project",
	Long: `A bot that uses large language models to act as a collaborator on a git project.

It can be used to:
* Monitor a repository for open issues, and generate LLM prompts according to the issue details
* Read an LLM response and process it into a new git commit and code change request on the version control server
`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		selfHandle := viper.GetString("handle")
		selfEmail := viper.GetString("email")
		repoDomain := viper.GetString("repo-domain")
		repoHandle := viper.GetString("repo-handle")
		repoName := viper.GetString("repo-name")
		githubToken := viper.GetString("github-token")
		localRepoPath := viper.GetString("local-repo-path")
		promptPath := viper.GetString("prompt-path")
		promptToClipboard := viper.GetBool("prompt-to-clipboard")
		responsePath := viper.GetString("response-path")

		/*
			log, err := zap.NewProduction()
			if err != nil {
				panic(err)
			}
		*/

		log := zap.L()

		author := vc.Author{
			Email:  selfEmail,
			Handle: selfHandle,
			Token:  githubToken,
		}
		repo := vc.Repository{
			LocalPath:  localRepoPath,
			HostDomain: repoDomain,
			Name:       repoName,
			Owner: vc.Author{
				Handle: repoHandle,
			},
		}
		p, err := pullpal.NewPullPal(cmd.Context(), log.Named("pullpal"), author, repo)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		// TODO this loop breaks on the second iteration due to a weird git state or something
		for {
			var input string
			fmt.Println("Press 'enter' when ready to select issue. Type 'exit' to exit.")
			fmt.Scanln(&input)
			if input == "exit" {
				break
			}

			var issue vc.Issue
			var changeRequest llm.CodeChangeRequest
			if promptToClipboard {
				issue, changeRequest, err = p.PickIssueToClipboard(promptPath)
				if err != nil {
					if !errors.Is(err, pullpal.IssueNotFound) {
						fmt.Println("error selecting issue and/or generating prompt", err)
						return
					} else {
						fmt.Println("No issues found. Proceeding to parse prompt")
					}
				} else {
					fmt.Printf("Picked issue and copied prompt to clipboard. Issue #%s\n", issue.ID)
				}
			} else {
				issue, changeRequest, err = p.PickIssueToFile(promptPath)
				if err != nil {
					if !errors.Is(err, pullpal.IssueNotFound) {
						fmt.Println("error selecting issue and/or generating prompt", err)
						return
					}
					fmt.Println("No issues found. Proceeding to parse prompt")
				} else {
					fmt.Printf("Picked issue and copied prompt to clipboard. Issue #%s. Prompt location %s\n", issue.ID, promptPath)
				}
			}

			fmt.Printf("\nInsert LLM response into response file: %s", responsePath)

			fmt.Println("Press 'enter' when ready to parse response. Enter 'skip' to skip response parsing. Enter 'exit' to exit.")
			fmt.Scanln(&input)
			if input == "exit" {
				break
			}
			if input == "skip" {
				fmt.Println()
				continue
			}

			prURL, err := p.ProcessResponseFromFile(changeRequest, responsePath)
			if err != nil {
				fmt.Println("error parsing LLM response and/or making version control changes", err)
				return
			}

			fmt.Printf("Successfully opened a code change request. Link: %s\n", prURL)
		}

		fmt.Println("Done. Thank you!")
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

var cfgFile string

func init() {
	cobra.OnInitialize(initConfig)

	// TODO make config values requried
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.pull-pal.yaml)")

	rootCmd.PersistentFlags().StringP("handle", "u", "HANDLE", "handle to use for version control actions")
	rootCmd.PersistentFlags().StringP("email", "e", "EMAIL", "email to use for version control actions")
	rootCmd.PersistentFlags().StringP("repo-domain", "d", "github.com", "domain for version control server")
	rootCmd.PersistentFlags().StringP("repo-handle", "o", "REPO-HANDLE", "handle of repository's owner on version control server")
	rootCmd.PersistentFlags().StringP("repo-name", "n", "REPO-NAME", "name of repository on version control server")
	rootCmd.PersistentFlags().StringP("github-token", "t", "GITHUB TOKEN", "token for authenticating Github actions")
	rootCmd.PersistentFlags().StringP("local-repo-path", "l", "/tmp/pullpalrepo/", "path where pull pal will check out a local copy of the repository")
	rootCmd.PersistentFlags().BoolP("prompt-to-clipboard", "c", false, "whether to copy LLM prompt to clipboard rather than using a file")
	rootCmd.PersistentFlags().StringP("prompt-path", "p", "./path/to/prompt.txt", "path where pull pal will write the llm prompt")
	rootCmd.PersistentFlags().StringP("response-path", "r", "./path/to/response.txt", "path where pull pal will read the llm response from")

	viper.BindPFlag("handle", rootCmd.PersistentFlags().Lookup("handle"))
	viper.BindPFlag("email", rootCmd.PersistentFlags().Lookup("email"))
	viper.BindPFlag("repo-domain", rootCmd.PersistentFlags().Lookup("repo-domain"))
	viper.BindPFlag("repo-handle", rootCmd.PersistentFlags().Lookup("repo-handle"))
	viper.BindPFlag("repo-name", rootCmd.PersistentFlags().Lookup("repo-name"))
	viper.BindPFlag("github-token", rootCmd.PersistentFlags().Lookup("github-token"))
	viper.BindPFlag("local-repo-path", rootCmd.PersistentFlags().Lookup("local-repo-path"))
	viper.BindPFlag("prompt-to-clipboard", rootCmd.PersistentFlags().Lookup("prompt-to-clipboard"))
	viper.BindPFlag("prompt-path", rootCmd.PersistentFlags().Lookup("prompt-path"))
	viper.BindPFlag("response-path", rootCmd.PersistentFlags().Lookup("response-path"))
}

func initConfig() {
	if cfgFile != "" {
		fmt.Println("cfg file exists")
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".pull-pal")
	}

	// TODO figure out how to get env variables to work
	viper.SetEnvPrefix("pullpal")
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}

}
