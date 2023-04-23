package cmd

import (
	"fmt"
	"os"

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
		responsePath := viper.GetString("response-path")

		log, err := zap.NewProduction()
		if err != nil {
			panic(err)
		}

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
			log.Error("error creating new pull pal", zap.Error(err))
			return
		}
		log.Info("Successfully initialized pull pal")

		issue, changeRequest, err := p.PickIssueToFile(promptPath)
		if err != nil {
			log.Error("error selecting issue and/or generating prompt", zap.Error(err))
			return
		}

		log.Info("Picked issue and created prompt", zap.String("issue ID", issue.ID), zap.String("prompt location", promptPath))

		log.Info("Insert LLM response into response file", zap.String("response location", responsePath))

		var input string
		log.Info("Press 'enter' when done.")
		fmt.Scanln(&input)

		prURL, err := p.ProcessResponseFromFile(changeRequest, responsePath)
		if err != nil {
			log.Error("error parsing LLM response and/or making version control changes", zap.Error(err))
			return
		}

		log.Info("Successfully opened a code change request", zap.String("Github PR link", prURL))
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
	rootCmd.PersistentFlags().StringP("prompt-path", "p", "./path/to/prompt.txt", "path where pull pal will write the llm prompt")
	rootCmd.PersistentFlags().StringP("response-path", "r", "./path/to/response.txt", "path where pull pal will read the llm response from")

	viper.BindPFlag("handle", rootCmd.PersistentFlags().Lookup("handle"))
	viper.BindPFlag("email", rootCmd.PersistentFlags().Lookup("email"))
	viper.BindPFlag("repo-domain", rootCmd.PersistentFlags().Lookup("repo-domain"))
	viper.BindPFlag("repo-handle", rootCmd.PersistentFlags().Lookup("repo-handle"))
	viper.BindPFlag("repo-name", rootCmd.PersistentFlags().Lookup("repo-name"))
	viper.BindPFlag("github-token", rootCmd.PersistentFlags().Lookup("github-token"))
	viper.BindPFlag("local-repo-path", rootCmd.PersistentFlags().Lookup("local-repo-path"))
	viper.BindPFlag("prompt-path", rootCmd.PersistentFlags().Lookup("prompt-path"))
	viper.BindPFlag("response-path", rootCmd.PersistentFlags().Lookup("response-path"))
}

func initConfig() {
	fmt.Println("init")
	if cfgFile != "" {
		fmt.Println("cfg file exists")
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		fmt.Println("cfg file empty")
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
