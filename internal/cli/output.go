package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/twinkle-apps/cli/internal/api"
)

// Styles for terminal output
var (
	dimStyle     = lipgloss.NewStyle().Faint(true)
	successStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
)

// Status prints a dimmed status message (for in-progress operations)
func Status(w io.Writer, msg string) {
	fmt.Fprintln(w, dimStyle.Render(msg))
}

// Statusf prints a formatted dimmed status message
func Statusf(w io.Writer, format string, args ...interface{}) {
	Status(w, fmt.Sprintf(format, args...))
}

// Success prints a green checkmark followed by a message
func Success(w io.Writer, msg string) {
	checkmark := successStyle.Render("✓")
	fmt.Fprintf(w, "%s %s\n", checkmark, successStyle.Render(msg))
}

// Successf prints a formatted success message with checkmark
func Successf(w io.Writer, format string, args ...interface{}) {
	Success(w, fmt.Sprintf(format, args...))
}

// MaskSecret masks all but the last `show` characters of a secret
// Example: MaskSecret("ABCD1234EFGH", 4) returns "●●●●●●●●EFGH"
func MaskSecret(secret string, show int) string {
	if len(secret) <= show {
		return secret
	}
	masked := strings.Repeat("●", len(secret)-show)
	return masked + secret[len(secret)-show:]
}

// Done prints the completion time in a human-friendly format
func Done(w io.Writer, elapsed time.Duration) {
	fmt.Fprintf(w, "Done in %.1fs\n", elapsed.Seconds())
}

// VerboseStatus prints a status with timing information (for verbose mode)
func VerboseStatus(w io.Writer, msg string, elapsed time.Duration) {
	fmt.Fprintln(w, dimStyle.Render(fmt.Sprintf("%s (%.1fs)", msg, elapsed.Seconds())))
}

func renderOutput(cmd *cobra.Command, jsonOut bool, verbose bool, payload interface{}) error {
	if jsonOut {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	}

	switch value := payload.(type) {
	case api.BuildResponse:
		printBuildResponse(cmd, value, verbose)
	case api.BuildUploadCompleteResponse:
		printUploadComplete(cmd, value, verbose)
	default:
		return fmt.Errorf("unsupported output type %T", payload)
	}
	return nil
}

func printBuildResponse(cmd *cobra.Command, resp api.BuildResponse, verbose bool) {
	out := cmd.OutOrStdout()

	// Print success messages for completed builds
	switch resp.Build.Status {
	case "complete", "published", "available":
		Successf(out, "Build %d processed", resp.Build.ID)
	default:
		fmt.Fprintf(out, "Build %d\n", resp.Build.ID)
		Statusf(out, "  Status: %s", resp.Build.Status)
	}

	if verbose {
		// Verbose mode: show all details
		fmt.Fprintf(out, "  Version: %s\n", resp.Build.Version)
		fmt.Fprintf(out, "  Build Number: %s\n", resp.Build.BuildNumber)
		fmt.Fprintf(out, "  Updated: %s\n", resp.Build.UpdatedAt.Format(time.RFC3339))

		if resp.Build.Metadata != nil {
			fmt.Fprintf(out, "  Metadata:\n")
			if resp.Build.Metadata.BuildVersion != nil {
				fmt.Fprintf(out, "    Build Version: %s\n", *resp.Build.Metadata.BuildVersion)
			}
			if resp.Build.Metadata.BuildNumber != nil {
				fmt.Fprintf(out, "    Build Number: %s\n", *resp.Build.Metadata.BuildNumber)
			}
			if resp.Build.Metadata.BuildSize != nil {
				fmt.Fprintf(out, "    Build Size: %s\n", formatBytes(*resp.Build.Metadata.BuildSize))
			}
			if resp.Build.Metadata.MinimumSystemVersion != nil {
				fmt.Fprintf(out, "    Minimum System: %s\n", *resp.Build.Metadata.MinimumSystemVersion)
			}
			if resp.Build.Metadata.Signature != nil {
				fmt.Fprintf(out, "    Signature: %s\n", *resp.Build.Metadata.Signature)
			}
			if len(resp.Build.Metadata.ProcessingErrors) > 0 {
				fmt.Fprintf(out, "    Processing Errors: %s\n", formatKeys(resp.Build.Metadata.ProcessingErrors))
			}
		}
	}

	// Appcast info
	if resp.Appcast.Status == "published" {
		Successf(out, "Feed updated: %s", resp.Appcast.FeedURL)
	} else {
		// Show feed status even in non-verbose mode
		Statusf(out, "Awaiting manual publication (status: %s)", resp.Appcast.Status)
		if verbose {
			fmt.Fprintf(out, "  Message: %s\n", resp.Appcast.Message)
			fmt.Fprintf(out, "  Feed URL: %s\n", resp.Appcast.FeedURL)
			if resp.Appcast.PublishedAt != nil {
				fmt.Fprintf(out, "  Published At: %s\n", resp.Appcast.PublishedAt.Format(time.RFC3339))
			}
			if resp.Appcast.URL != nil {
				fmt.Fprintf(out, "  URL: %s\n", *resp.Appcast.URL)
			}
		}
	}
}

func printUploadComplete(cmd *cobra.Command, resp api.BuildUploadCompleteResponse, verbose bool) {
	out := cmd.OutOrStdout()
	Success(out, "Upload complete")
	if verbose {
		fmt.Fprintf(out, "  Build ID: %d\n", resp.BuildID.Int())
		fmt.Fprintf(out, "  Status URL: %s\n", resp.StatusURL)
		fmt.Fprintf(out, "  Wait URL: %s\n", resp.WaitURL)
	}
}

func formatKeys(values map[string]interface{}) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return strings.Join(keys, ", ")
}

// formatBytes formats a byte count as a human-readable string
func formatBytes(bytes int) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/KB)
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}
