package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var researchCmd = &cobra.Command{
	Use:   "research",
	Short: "Deep research workflow",
}

var researchSendCmd = &cobra.Command{
	Use:   "send [prompt]",
	Short: "Submit a deep research task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		prompt := args[0]
		model := resolveModel()
		plan, err := c.CreateAndStartDeepResearch(ctx, prompt, model)
		if err != nil {
			return err
		}

		fmt.Println("Deep Research task submitted")
		if plan.Title != "" {
			fmt.Printf("  Title:  %s\n", plan.Title)
		}
		if plan.ETAText != "" {
			fmt.Printf("  ETA:    %s\n", plan.ETAText)
		}
		if len(plan.Steps) > 0 {
			fmt.Println("  Steps:")
			for _, step := range plan.Steps {
				fmt.Printf("    - %s\n", step)
			}
		}
		fmt.Printf("\n  Chat ID: %s\n", plan.Cid)
		fmt.Printf("\n  Use 'research check %s' to check progress\n", plan.Cid)
		fmt.Printf("  Use 'research get %s' to fetch result\n", plan.Cid)
		return nil
	},
}

var researchCheckCmd = &cobra.Command{
	Use:   "check [chat_id]",
	Short: "Check deep research progress",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		cid := args[0]
		done, textLen, err := c.CheckDeepResearch(ctx, cid)
		if err != nil {
			return err
		}

		if textLen == 0 {
			fmt.Println("  Status: waiting (no response yet)")
		} else {
			status := "in progress"
			if done {
				status = "done"
			}
			fmt.Printf("  Status: %s\n", status)
			fmt.Printf("  Response length: %d chars\n", textLen)
			if done {
				fmt.Printf("\n  Use 'research get %s' to retrieve the full result.\n", cid)
			}
		}
		return nil
	},
}

var researchGetOutput string

var researchGetCmd = &cobra.Command{
	Use:   "get [chat_id]",
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

		if researchGetOutput != "" {
			dir := filepath.Dir(researchGetOutput)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
			if err := os.WriteFile(researchGetOutput, []byte(result), 0644); err != nil {
				return err
			}
			fmt.Printf("Saved research result to %s (%d sources)\n", researchGetOutput, len(sources))
		} else {
			fmt.Println(result)
		}
		return nil
	},
}

func init() {
	researchGetCmd.Flags().StringVar(&researchGetOutput, "output", "", "Write result to file instead of stdout")

	researchCmd.AddCommand(researchSendCmd)
	researchCmd.AddCommand(researchCheckCmd)
	researchCmd.AddCommand(researchGetCmd)
}
