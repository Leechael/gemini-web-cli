package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

var (
	researchListCount int
	researchListJSON  bool
)

var researchCmd = &cobra.Command{
	Use:   "research",
	Short: "Deep research utilities",
	Long:  "Submit deep research tasks and list completed reports.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		}
		return cmd.Help()
	},
}

var researchRunCmd = &cobra.Command{
	Use:   "run <prompt>",
	Short: "Submit a deep research task",
	Args:  cobra.ExactArgs(1),
	RunE:  runResearchRun,
}

var researchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List completed deep research reports from /library",
	Args:  cobra.NoArgs,
	RunE:  runResearchList,
}

func runResearchRun(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	prompt := args[0]
	model := resolveModelForClient(ctx, c)
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
}

func runResearchList(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	reports, err := c.ListResearchReports(ctx, researchListCount)
	if err != nil {
		return err
	}
	if researchListJSON {
		out, err := json.Marshal(reports)
		if err != nil {
			return err
		}
		fmt.Println(string(out))
		return nil
	}
	for _, report := range reports {
		fmt.Printf("Cid:    %s\n", report.Cid)
		fmt.Printf("Title:  %s\n", report.Title)
		if report.CreatedAt != 0 {
			fmt.Printf("Date:   %s\n", time.Unix(report.CreatedAt, 0).UTC().Format("2006-01-02 15:04 UTC"))
		}
		fmt.Printf("Report: %s\n", report.ReportID)
		fmt.Println("---")
	}
	return nil
}

func init() {
	researchListCmd.Flags().IntVar(&researchListCount, "count", 4, "max reports to return")
	researchListCmd.Flags().BoolVar(&researchListJSON, "json", false, "Output JSON")
	researchCmd.AddCommand(researchRunCmd, researchListCmd)
}
