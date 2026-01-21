package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/twinkle-apps/cli/internal/api"
)

func renderOutput(cmd *cobra.Command, jsonOut bool, payload interface{}) error {
	if jsonOut {
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(payload)
	}

	switch value := payload.(type) {
	case api.BuildResponse:
		printBuildResponse(cmd, value)
	case api.BuildUploadCompleteResponse:
		printUploadComplete(cmd, value)
	default:
		return fmt.Errorf("unsupported output type %T", payload)
	}
	return nil
}

func printBuildResponse(cmd *cobra.Command, resp api.BuildResponse) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Build %d\n", resp.Build.ID)
	fmt.Fprintf(out, "  Status: %s\n", resp.Build.Status)
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
			fmt.Fprintf(out, "    Build Size: %d\n", *resp.Build.Metadata.BuildSize)
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

	fmt.Fprintf(out, "Appcast\n")
	fmt.Fprintf(out, "  Status: %s\n", resp.Appcast.Status)
	fmt.Fprintf(out, "  Message: %s\n", resp.Appcast.Message)
	fmt.Fprintf(out, "  Feed URL: %s\n", resp.Appcast.FeedURL)
	if resp.Appcast.PublishedAt != nil {
		fmt.Fprintf(out, "  Published At: %s\n", resp.Appcast.PublishedAt.Format(time.RFC3339))
	}
	if resp.Appcast.URL != nil {
		fmt.Fprintf(out, "  URL: %s\n", *resp.Appcast.URL)
	}
}

func printUploadComplete(cmd *cobra.Command, resp api.BuildUploadCompleteResponse) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Upload complete\n")
	fmt.Fprintf(out, "  Build ID: %d\n", resp.BuildID.Int())
	fmt.Fprintf(out, "  Status URL: %s\n", resp.StatusURL)
	fmt.Fprintf(out, "  Wait URL: %s\n", resp.WaitURL)
}

func formatKeys(values map[string]interface{}) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return strings.Join(keys, ", ")
}
