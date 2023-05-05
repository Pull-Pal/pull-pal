package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var debugLLMCmd = &cobra.Command{
	Use:   "debug-llm",
	Short: "debug llm functionality",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := getConfig()

		p, err := getPullPal(cmd.Context(), cfg)
		if err != nil {
			fmt.Println("error creating new pull pal", err)
			return
		}
		fmt.Println("Successfully initialized pull pal")

		err = p.DebugLLM()
		if err != nil {
			fmt.Println("err debugging prompts", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(debugLLMCmd)
}
