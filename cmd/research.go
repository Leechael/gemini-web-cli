package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

var researchCmd = &cobra.Command{
	Use:   "research [prompt]",
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
		fmt.Printf("\n  Use 'progress %s' to check progress\n", plan.Cid)
		fmt.Printf("  Use 'report %s' to fetch result\n", plan.Cid)
		return nil
	},
}
