package cmd

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var expandPromptJSON bool

var expandPromptCmd = &cobra.Command{
	Use:   "expand-prompt <text>",
	Short: "Expand a media prompt into alternative descriptions",
	Long:  "Calls the media prompt expansion RPC to generate alternative phrasings for image, music, or video prompts.",
	Args:  cobra.ExactArgs(1),
	RunE:  runExpandPrompt,
}

func runExpandPrompt(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	variations, err := c.ExpandMediaPrompt(ctx, args[0])
	if err != nil {
		return err
	}

	if expandPromptJSON {
		if variations == nil {
			variations = []string{}
		}
		out, err := json.MarshalIndent(variations, "", "  ")
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}

	if len(variations) == 0 {
		fmt.Println("no variations returned")
		return nil
	}
	for i, variation := range variations {
		fmt.Printf("%d. %s\n", i+1, variation)
	}
	return nil
}

func init() {
	expandPromptCmd.Flags().BoolVar(&expandPromptJSON, "json", false, "Output JSON")
}
