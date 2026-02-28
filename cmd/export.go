package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/trailblaze/claude-replay/internal/export"
	"github.com/trailblaze/claude-replay/internal/session"
)

var (
	exportMode   string
	exportFormat string
	exportOutput string
	exportWidth  int
	exportHeight int
)

var exportCmd = &cobra.Command{
	Use:   "export <session>",
	Short: "Export a session as an asciinema recording",
	Long:  "Export a session as an asciinema .cast file, with optional conversion to GIF or MP4",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]

		// Find the session
		path, err := session.FindSessionByID(claudeDir, query)
		if err != nil {
			return fmt.Errorf("finding session: %w", err)
		}

		// Load it
		sess, err := session.LoadSession(path)
		if err != nil {
			return fmt.Errorf("loading session: %w", err)
		}

		if len(sess.Turns) == 0 {
			return fmt.Errorf("session has no turns")
		}

		// Build options
		opts := export.Options{
			TimingMode: export.TimingMode(exportMode),
			Width:      exportWidth,
			Height:     exportHeight,
			Format:     exportFormat,
		}

		// Determine output path
		if exportOutput == "" {
			slug := sess.Slug
			if slug == "" && len(sess.ID) > 8 {
				slug = sess.ID[:8]
			}
			exportOutput = slug + "." + exportFormat
		}

		// Generate .cast file
		castPath := exportOutput
		if !strings.HasSuffix(castPath, ".cast") && opts.Format == "cast" {
			// Output is already the right path
		} else if opts.Format != "cast" {
			castPath = strings.TrimSuffix(exportOutput, "."+opts.Format) + ".cast"
		}

		opts.Output = castPath

		fmt.Printf("Exporting session: %s\n", sess.Slug)
		fmt.Printf("  Turns: %d\n", len(sess.Turns))
		fmt.Printf("  Mode: %s\n", opts.TimingMode)
		fmt.Printf("  Output: %s\n", castPath)

		if err := export.GenerateCast(sess, opts); err != nil {
			return fmt.Errorf("generating cast: %w", err)
		}

		fmt.Printf("  Done: %s\n", export.FormatCastInfo(castPath))

		// Convert if needed
		if opts.Format == "gif" {
			gifPath := exportOutput
			if err := export.ConvertToGif(castPath, gifPath); err != nil {
				fmt.Printf("  Note: %v\n", err)
				fmt.Printf("  You can convert manually: agg %s %s\n", castPath, gifPath)
			}
		} else if opts.Format == "mp4" {
			gifPath := strings.TrimSuffix(exportOutput, ".mp4") + ".gif"
			if err := export.ConvertToGif(castPath, gifPath); err != nil {
				fmt.Printf("  Note: %v\n", err)
			} else {
				if err := export.ConvertToMP4(gifPath, exportOutput); err != nil {
					fmt.Printf("  Note: %v\n", err)
				}
			}
		}

		return nil
	},
}

func init() {
	exportCmd.Flags().StringVar(&exportMode, "mode", "compressed", "timing mode: realtime, compressed, fast, instant")
	exportCmd.Flags().StringVar(&exportFormat, "format", "cast", "output format: cast, gif, mp4")
	exportCmd.Flags().StringVarP(&exportOutput, "output", "o", "", "output file path")
	exportCmd.Flags().IntVar(&exportWidth, "width", 120, "terminal width")
	exportCmd.Flags().IntVar(&exportHeight, "height", 40, "terminal height")

	rootCmd.AddCommand(exportCmd)
}
