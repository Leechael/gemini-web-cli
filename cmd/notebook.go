package cmd

import (
	"context"
	"fmt"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/spf13/cobra"
)

var notebookCmd = &cobra.Command{
	Use:   "notebook",
	Short: "Manage Gemini notebooks",
}

var notebookCreateCmd = &cobra.Command{
	Use:   "create [title]",
	Short: "Create a notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		resource, err := c.CreateNotebook(ctx, args[0])
		if err != nil {
			return err
		}
		fmt.Println(resource)
		fmt.Fprintf(cmd.ErrOrStderr(), "Created notebook %q as %s\n", args[0], resource)
		return nil
	},
}

var notebookGetCmd = &cobra.Command{
	Use:   "get [notebook_id]",
	Short: "Show notebook details and sources",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		nb, err := c.GetNotebook(ctx, args[0])
		if err != nil {
			return err
		}
		if nb == nil {
			fmt.Printf("Notebook %s not found\n", args[0])
			return fmt.Errorf("notebook not found")
		}

		title := nb.Title
		if nb.Emoji != "" {
			title = nb.Emoji + " " + title
		}
		fmt.Printf("%s\n  %s\n", title, nb.ResourceName)
		if len(nb.Sources) == 0 {
			fmt.Println("  No sources")
			return nil
		}
		fmt.Printf("  %d source(s):\n", len(nb.Sources))
		for _, src := range nb.Sources {
			fmt.Printf("    %s (%s)\n      %s\n", src.FileName, src.MimeType, src.ResourceName)
		}
		return nil
	},
}

var notebookChatsCmd = &cobra.Command{
	Use:   "chats [notebook_id]",
	Short: "List chats in a notebook",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		items, err := c.ListNotebookChats(ctx, args[0])
		if err != nil {
			return err
		}
		if len(items) == 0 {
			fmt.Println("No chats in this notebook.")
			return nil
		}
		for _, it := range items {
			fmt.Printf("%s  %s  %s\n", it.Cid, it.Title, it.UpdatedAt)
		}
		return nil
	},
}

var notebookAddSourceCmd = &cobra.Command{
	Use:   "add-source [notebook_id] [file...]",
	Short: "Upload files and attach them to a notebook as sources",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		notebookID := args[0]
		for _, f := range args[1:] {
			fmt.Fprintf(cmd.ErrOrStderr(), "Uploading %s...\n", f)
			u, err := c.UploadFile(ctx, f)
			if err != nil {
				return fmt.Errorf("upload %s failed: %w", f, err)
			}
			nb, err := c.AddNotebookSource(ctx, notebookID, u.FileName, u.MimeType, u.ID)
			if err != nil {
				return fmt.Errorf("attach %s failed: %w", f, err)
			}
			if nb == nil {
				return fmt.Errorf("attach %s failed: empty notebook response", f)
			}
			fmt.Printf("Attached %s (%d source(s) total)\n", u.FileName, len(nb.Sources))
		}
		return nil
	},
}

var notebookAddURLCmd = &cobra.Command{
	Use:   "add-url [notebook_id] [url...]",
	Short: "Attach web URLs to a notebook as sources",
	Args:  cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		notebookID := args[0]
		for _, u := range args[1:] {
			if !client.IsHTTPURL(u) {
				return fmt.Errorf("invalid URL %q", u)
			}
			nb, err := c.AddNotebookURLSource(ctx, notebookID, u)
			if err != nil {
				return fmt.Errorf("attach %s failed: %w", u, err)
			}
			if nb == nil {
				return fmt.Errorf("attach %s failed: empty notebook response", u)
			}
			fmt.Printf("Attached %s (%d source(s) total)\n", u, len(nb.Sources))
		}
		return nil
	},
}

var notebookRemoveSourceCmd = &cobra.Command{
	Use:   "remove-source [source_resource]",
	Short: "Remove a source from a notebook (notebooks/<uuid>/sources/<sid>)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		c, jsonCookies, err := initClient(ctx)
		if err != nil {
			return err
		}
		defer cleanup(c, jsonCookies)

		if err := c.RemoveNotebookSource(ctx, args[0]); err != nil {
			return err
		}
		fmt.Printf("Removed %s\n", args[0])
		return nil
	},
}

func init() {
	notebookCmd.AddCommand(
		notebookCreateCmd,
		notebookGetCmd,
		notebookChatsCmd,
		notebookAddSourceCmd,
		notebookAddURLCmd,
		notebookRemoveSourceCmd,
	)
}
