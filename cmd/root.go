package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mobyvb/pull-pal/pullpal"
	"github.com/mobyvb/pull-pal/vc"
	"github.com/sashabaranov/go-openai"
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
	repos []string

	// local paths
	localRepoPath string

	// program settings
	usersToListenTo     []string
	requiredIssueLabels []string
	waitDuration        time.Duration
	debugDir            string
	queueSize           int
}

func getConfig() config {
	return config{
		selfHandle:  viper.GetString("handle"),
		selfEmail:   viper.GetString("email"),
		githubToken: viper.GetString("github-token"),
		openAIToken: viper.GetString("open-ai-token"),

		repos: viper.GetStringSlice("repos"),

		localRepoPath: viper.GetString("local-repo-path"),

		usersToListenTo:     viper.GetStringSlice("users-to-listen-to"),
		requiredIssueLabels: viper.GetStringSlice("required-issue-labels"),
		waitDuration:        viper.GetDuration("wait-duration"),
		debugDir:            viper.GetString("debug-dir"),
		queueSize:           viper.GetInt("queue-size"),
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
	listIssueOptions := vc.ListIssueOptions{
		Handles: cfg.usersToListenTo,
		Labels:  cfg.requiredIssueLabels,
	}
	// TODO make model configurable
	ppCfg := pullpal.Config{
		WaitDuration:     cfg.waitDuration,
		QueueSize:        cfg.queueSize,
		LocalRepoPath:    cfg.localRepoPath,
		Repos:            cfg.repos,
		Self:             author,
		ListIssueOptions: listIssueOptions,
		// TODO configurable model
		Model:       openai.GPT4,
		OpenAIToken: cfg.openAIToken,
		DebugDir:    cfg.debugDir,
	}
	p, err := pullpal.NewPullPal(ctx, log.Named("pullpal"), ppCfg)

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

	rootCmd.PersistentFlags().StringSliceP("repos", "r", []string{}, "a list of git repositories that Pull Pal will monitor")

	rootCmd.PersistentFlags().StringP("local-repo-path", "l", "/tmp/pullpalrepo", "local path to check out ephemeral repository in")

	rootCmd.PersistentFlags().StringSliceP("users-to-listen-to", "a", []string{}, "a list of Github users that Pull Pal will respond to")
	rootCmd.PersistentFlags().StringSliceP("required-issue-labels", "i", []string{}, "a list of labels that are required for Pull Pal to select an issue")
	rootCmd.PersistentFlags().Duration("wait-time", 30*time.Second, "the amount of time Pull Pal should wait when no issues or comments are found to address")
	rootCmd.PersistentFlags().StringP("debug-dir", "d", "", "the path to use for the pull pal debug directory")
	rootCmd.PersistentFlags().Int("queue-size", 10, "the size of the task queue for each repo")

	viper.BindPFlag("handle", rootCmd.PersistentFlags().Lookup("handle"))
	viper.BindPFlag("email", rootCmd.PersistentFlags().Lookup("email"))
	viper.BindPFlag("github-token", rootCmd.PersistentFlags().Lookup("github-token"))
	viper.BindPFlag("open-ai-token", rootCmd.PersistentFlags().Lookup("open-ai-token"))

	viper.BindPFlag("repos", rootCmd.PersistentFlags().Lookup("repos"))

	viper.BindPFlag("local-repo-path", rootCmd.PersistentFlags().Lookup("local-repo-path"))

	viper.BindPFlag("users-to-listen-to", rootCmd.PersistentFlags().Lookup("users-to-listen-to"))
	viper.BindPFlag("required-issue-labels", rootCmd.PersistentFlags().Lookup("required-issue-labels"))
	viper.BindPFlag("wait-time", rootCmd.PersistentFlags().Lookup("wait-time"))
	viper.BindPFlag("debug-dir", rootCmd.PersistentFlags().Lookup("debug-dir"))
	viper.BindPFlag("queue-size", rootCmd.PersistentFlags().Lookup("queue-size"))
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
