package cmd

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Check login status (alias for inspect)",
	RunE:  inspectCmd.RunE,
}
