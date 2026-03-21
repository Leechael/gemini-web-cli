package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Leechael/gemini-web-cli/internal/client"
	"github.com/Leechael/gemini-web-cli/internal/types"
	"github.com/spf13/cobra"
)

var (
	askNoStream bool
	askFiles    []string
)

// textExtensions lists file extensions that should be inlined into the prompt.
var textExtensions = map[string]bool{
	".txt": true, ".md": true, ".markdown": true,
	".json": true, ".yaml": true, ".yml": true, ".toml": true,
	".xml": true, ".csv": true, ".tsv": true,
	".go": true, ".py": true, ".js": true, ".ts": true, ".jsx": true, ".tsx": true,
	".rs": true, ".c": true, ".cpp": true, ".h": true, ".hpp": true,
	".java": true, ".kt": true, ".swift": true, ".rb": true, ".php": true,
	".sh": true, ".bash": true, ".zsh": true, ".fish": true,
	".sql": true, ".html": true, ".css": true, ".scss": true,
	".env": true, ".ini": true, ".cfg": true, ".conf": true,
	".log": true, ".diff": true, ".patch": true,
	".tex": true, ".rst": true, ".adoc": true,
}

func isTextFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return textExtensions[ext]
}

var askCmd = &cobra.Command{
	Use:   "ask [prompt]",
	Short: "Single-turn question (prefix with @Google to trigger search)",
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

		// Process --file: text files inlined, binary files uploaded via resumable protocol
		var uploads []*client.UploadResult
		for _, f := range askFiles {
			if isTextFile(f) {
				content, err := os.ReadFile(f)
				if err != nil {
					return fmt.Errorf("reading %s: %w", f, err)
				}
				name := filepath.Base(f)
				prompt = fmt.Sprintf("<file name=%q>\n%s\n</file>\n\n%s", name, string(content), prompt)
				fmt.Fprintf(cmd.ErrOrStderr(), "Attached %s (%d bytes, inlined)\n", name, len(content))
			} else {
				fmt.Fprintf(cmd.ErrOrStderr(), "Uploading %s...\n", f)
				u, err := c.UploadFile(ctx, f)
				if err != nil {
					return fmt.Errorf("upload %s failed: %w", f, err)
				}
				uploads = append(uploads, u)
				fmt.Fprintf(cmd.ErrOrStderr(), "Uploaded %s (ID: %s)\n", u.FileName, u.ID)
			}
		}

		if askNoStream {
			var output *types.ModelOutput
			if len(uploads) > 0 {
				output, err = c.GenerateContentWithFiles(ctx, prompt, uploads, model)
			} else {
				output, err = c.GenerateContent(ctx, prompt, model)
			}
			if err != nil {
				return err
			}
			fmt.Println(output.Text)
			printImages(output)
			printVideos(output)
			printMedia(output)
			printChatID(output)
		} else {
			var output *types.ModelOutput
			if len(uploads) > 0 {
				output, err = c.GenerateContentStreamWithFiles(ctx, prompt, uploads, model, func(out *types.ModelOutput) {
					if out.TextDelta != "" {
						fmt.Print(out.TextDelta)
					}
				})
			} else {
				output, err = c.GenerateContentStream(ctx, prompt, model, func(out *types.ModelOutput) {
					if out.TextDelta != "" {
						fmt.Print(out.TextDelta)
					}
				})
			}
			if err != nil {
				return err
			}
			if output != nil {
				fmt.Println()
				printImages(output)
				printChatID(output)
			}
		}
		return nil
	},
}

func init() {
	askCmd.Flags().BoolVar(&askNoStream, "no-stream", false, "Wait for complete response")
	askCmd.Flags().StringArrayVarP(&askFiles, "file", "f", nil, "Attach file(s) (can be specified multiple times)")
}

func printImages(output *types.ModelOutput) {
	if output == nil || len(output.Images) == 0 {
		return
	}
	var web, gen []types.Image
	for _, img := range output.Images {
		if img.Generated {
			gen = append(gen, img)
		} else {
			web = append(web, img)
		}
	}
	if len(web) > 0 {
		fmt.Println("\n---\nImages:")
		for i, img := range web {
			title := ""
			if img.Title != "" {
				title = "  " + img.Title
			}
			fmt.Printf("  %d) %s%s\n", i+1, img.URL, title)
		}
	}
	if len(gen) > 0 {
		fmt.Println("\n---\nGenerated images:")
		for i, img := range gen {
			fmt.Printf("  %d) %s\n", i+1, img.URL)
		}
	}
}

func printVideos(output *types.ModelOutput) {
	if output == nil || len(output.Videos) == 0 {
		return
	}
	fmt.Println("\n---\nGenerated videos:")
	for i, vid := range output.Videos {
		fmt.Printf("  %d) %s\n", i+1, vid.URL)
		if vid.Thumbnail != "" {
			fmt.Printf("     Thumbnail: %s\n", vid.Thumbnail)
		}
	}
}

func printMedia(output *types.ModelOutput) {
	if output == nil || len(output.Media) == 0 {
		return
	}
	fmt.Println("\n---\nGenerated media:")
	for i, m := range output.Media {
		fmt.Printf("  %d)", i+1)
		if m.MP3URL != "" {
			fmt.Printf(" MP3: %s", m.MP3URL)
		}
		if m.MP4URL != "" {
			fmt.Printf(" MP4: %s", m.MP4URL)
		}
		fmt.Println()
	}
}

func printChatID(output *types.ModelOutput) {
	if output != nil && len(output.Metadata) > 0 && output.Metadata[0] != "" {
		fmt.Printf("\n---\nChat ID: %s\n", output.Metadata[0])
	}
}
