package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/mobyvb/pull-pal/pullpal"
	"github.com/mobyvb/pull-pal/vc"
	"go.uber.org/zap"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// todo: some of this config definition/usage can be moved to other packages
type config struct {
	// bot credentials + github info
	selfHandle  string
	selfEmail   string
	githubToken string
	openAIToken string

	// remote repo info
	repoDomain string
	repoHandle string
	repoName   string

	// local paths
	localRepoPath string

	// program settings
	promptToClipboard   bool
	usersToListenTo     []string
	requiredIssueLabels []string
}

func getConfig() config {
	return config{
		selfHandle:  viper.GetString("handle"),
		selfEmail:   viper.GetString("email"),
		githubToken: viper.GetString("github-token"),
		openAIToken: viper.GetString("open-ai-token"),

		repoDomain: viper.GetString("repo-domain"),
		repoHandle: viper.GetString("repo-handle"),
		repoName:   viper.GetString("repo-name"),

		localRepoPath: viper.GetString("local-repo-path"),

		promptToClipboard:   viper.GetBool("prompt-to-clipboard"),
		usersToListenTo:     viper.GetStringSlice("users-to-listen-to"),
		requiredIssueLabels: viper.GetStringSlice("required-issue-labels"),
	}
}

func getPullPal(ctx context.Context, cfg config) (*pullpal.PullPal, error) {
	// TODO figure out debug logging
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	//log := zap.L()

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
	listIssueOptions := vc.ListIssueOptions{
		Handles: cfg.usersToListenTo,
		Labels:  cfg.requiredIssueLabels,
	}
	p, err := pullpal.NewPullPal(ctx, log.Named("pullpal"), listIssueOptions, author, repo, cfg.openAIToken)

	return p, err
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Short: "run an automated digital assitant to monitor and make code changes to a github repository",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		err = p.Run()
		if err != nil {
			fmt.Println("error running", err)
			return
		}
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
	rootCmd.PersistentFlags().StringP("github-token", "t", "GITHUB TOKEN", "token for authenticating Github actions")
	rootCmd.PersistentFlags().StringP("open-ai-token", "k", "OPENAI TOKEN", "token for authenticating OpenAI")

	rootCmd.PersistentFlags().StringP("repo-domain", "d", "github.com", "domain for version control server")
	rootCmd.PersistentFlags().StringP("repo-handle", "o", "REPO-HANDLE", "handle of repository's owner on version control server")
	rootCmd.PersistentFlags().StringP("repo-name", "n", "REPO-NAME", "name of repository on version control server")

	rootCmd.PersistentFlags().StringP("local-repo-path", "l", "/tmp/pullpallrepo", "local path to check out ephemeral repository in")

	rootCmd.PersistentFlags().StringSliceP("users-to-listen-to", "a", []string{}, "a list of Github users that Pull Pal will respond to")
	rootCmd.PersistentFlags().StringSliceP("required-issue-labels", "i", []string{}, "a list of labels that are required for Pull Pal to select an issue")

	viper.BindPFlag("handle", rootCmd.PersistentFlags().Lookup("handle"))
	viper.BindPFlag("email", rootCmd.PersistentFlags().Lookup("email"))
	viper.BindPFlag("github-token", rootCmd.PersistentFlags().Lookup("github-token"))
	viper.BindPFlag("open-ai-token", rootCmd.PersistentFlags().Lookup("open-ai-token"))

	viper.BindPFlag("repo-domain", rootCmd.PersistentFlags().Lookup("repo-domain"))
	viper.BindPFlag("repo-handle", rootCmd.PersistentFlags().Lookup("repo-handle"))
	viper.BindPFlag("repo-name", rootCmd.PersistentFlags().Lookup("repo-name"))

	viper.BindPFlag("local-repo-path", rootCmd.PersistentFlags().Lookup("local-repo-path"))

	viper.BindPFlag("users-to-listen-to", rootCmd.PersistentFlags().Lookup("users-to-listen-to"))
	viper.BindPFlag("required-issue-labels", rootCmd.PersistentFlags().Lookup("required-issue-labels"))
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
