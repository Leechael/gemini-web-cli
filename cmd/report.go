package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var reportOutput string

var reportCmd = &cobra.Command{
	Use:   "report [chat_id]",
	Short: "Get deep research result",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		cid := args[0]
		text, sources, err := c.GetDeepResearchResult(ctx, cid)
		if err != nil {
			return err
		}

		result := text
		if len(sources) > 0 {
			result += "\n\n---\n\n## References\n"
			for num := range sources {
				s := sources[num]
				result += fmt.Sprintf("[%d] [%s](%s)\n", num, s.Title, s.URL)
			}
		}

		if reportOutput != "" {
			dir := filepath.Dir(reportOutput)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(reportOutput, []byte(result), 0644); err != nil {
				return err
			}
			fmt.Printf("Saved research result to %s (%d sources)\n", reportOutput, len(sources))
		} else {
			fmt.Println(result)
		}
		return nil
	},
}

func init() {
	reportCmd.Flags().StringVar(&reportOutput, "output", "", "Write result to file instead of stdout")
}
