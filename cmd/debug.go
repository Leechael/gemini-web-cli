package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/spf13/cobra"
)

var (
	debugPayload    string
	debugSourcePath string
	debugSourceCid  string
	debugPretty     bool
)

var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Debugging utilities for protocol exploration",
	Long:  "Low-level utilities for exercising RPCs directly. Useful for verifying new RPC integrations or investigating protocol changes.",
}

var debugRPCCmd = &cobra.Command{
	Use:   "rpc <rpcID>",
	Short: "Call any batchexecute RPC and print raw response",
	Long: `Calls the specified RPC ID via the batchexecute endpoint and prints
the raw wrb.fr body (the inner JSON string).

Examples:
  # Call a no-arg RPC
  gemini-web-cli debug rpc otAQ7b

  # Call with payload
  gemini-web-cli debug rpc MaZiqc --payload '[13,null,[1,null,1]]'

  # Call within a chat context
  gemini-web-cli debug rpc hNvQHb --payload '["c_abc",10]' --source-cid c_abc

  # Pretty-print JSON output
  gemini-web-cli debug rpc cYRIkd --payload '["en"]' --pretty`,
	Args: cobra.ExactArgs(1),
	RunE: runDebugRPC,
}

func runDebugRPC(cmd *cobra.Command, args []string) error {
	rpcID := args[0]
	ctx := context.Background()

	c, jsonCookies, err := initClient(ctx)
	if err != nil {
		return err
	}
	defer cleanup(c, jsonCookies)

	var opts []client.RPCOpt
	switch {
	case debugSourceCid != "" && debugSourcePath != "":
		return fmt.Errorf("--source-cid and --source-path are mutually exclusive")
	case debugSourceCid != "":
		opts = append(opts, client.WithSourceCid(debugSourceCid))
	case debugSourcePath != "":
		opts = append(opts, client.WithSourcePath(debugSourcePath))
	}

	start := time.Now()
	body, rejectCode, err := c.CallRPC(ctx, rpcID, debugPayload, opts...)
	elapsed := time.Since(start)
	if err != nil {
		return fmt.Errorf("CallRPC failed: %w", err)
	}

	fmt.Fprintf(os.Stderr, "rpcID=%s rejectCode=%d bodyLen=%d elapsed=%s\n",
		rpcID, rejectCode, len(body), elapsed.Round(time.Millisecond))

	if debugPretty {
		var v any
		if jerr := json.Unmarshal(body, &v); jerr == nil {
			pretty, _ := json.MarshalIndent(v, "", "  ")
			fmt.Println(string(pretty))
			return nil
		}
		fmt.Fprintln(os.Stderr, "warning: body is not valid JSON, printing raw")
	}
	_, _ = os.Stdout.Write(body)
	if len(body) > 0 && body[len(body)-1] != '\n' {
		fmt.Println()
	}
	return nil
}

func init() {
	debugRPCCmd.Flags().StringVar(&debugPayload, "payload", "[]", "RPC payload as a JSON string")
	debugRPCCmd.Flags().StringVar(&debugSourcePath, "source-path", "", "Override source-path query param")
	debugRPCCmd.Flags().StringVar(&debugSourceCid, "source-cid", "", "Convenience: sets source-path to <appPath>/<cid>")
	debugRPCCmd.Flags().BoolVar(&debugPretty, "pretty", false, "Pretty-print the response if it's valid JSON")

	debugCmd.AddCommand(debugRPCCmd)
}
