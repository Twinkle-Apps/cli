//go:build debug

package cli

import (
	"time"

	"github.com/spf13/cobra"
)

func init() {
	// Register the demo command hook for debug builds
	registerDemoCommand = addDemoCommand
}

func addDemoCommand(parent *cobra.Command) {
	parent.AddCommand(newDemoCmd())
}

func newDemoCmd() *cobra.Command {
	var (
		fast   bool
		styles bool
	)

	cmd := &cobra.Command{
		Use:    "demo",
		Short:  "[DEBUG] Demonstrate styled terminal output",
		Long:   "Shows all available output styles and animations. Only available in debug builds.",
		Hidden: false,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			stderr := cmd.ErrOrStderr()

			if styles {
				return runStylesDemo(cmd)
			}

			return runAnimatedDemo(cmd, out, stderr, fast)
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Skip delays for quick testing")
	cmd.Flags().BoolVar(&styles, "styles", false, "Show all styles without animation")

	return cmd
}

func runStylesDemo(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()

	// Header
	cmd.Println("=== Styled Output Demo ===")
	cmd.Println()

	// Status messages (dimmed)
	cmd.Println("Status messages (dimmed/in-progress):")
	Status(out, "Preparing upload…")
	Status(out, "Uploading MyApp.zip…")
	Statusf(out, "Processing build %d…", 42)
	cmd.Println()

	// Success messages (green checkmark)
	cmd.Println("Success messages:")
	Success(out, "Build uploaded successfully")
	Successf(out, "Published to %d regions", 12)
	Success(out, "Feed updated: https://example.com/appcast.xml")
	cmd.Println()

	// Error messages
	cmd.Println("Error messages:")
	Error(out, "Build 42 failed")
	ErrorDetail(out, "signing: missing certificate")
	ErrorDetail(out, "version: Build number must be greater than 2022.01.01")
	cmd.Println()

	// Masked secrets
	cmd.Println("Masked secrets:")
	cmd.Printf("  Full key:   ABCD1234EFGH5678\n")
	cmd.Printf("  Masked (4): %s\n", MaskSecret("ABCD1234EFGH5678", 4))
	cmd.Printf("  Masked (8): %s\n", MaskSecret("ABCD1234EFGH5678", 8))
	cmd.Printf("  Short key:  %s\n", MaskSecret("ABC", 4))
	cmd.Println()

	// Timing
	cmd.Println("Timing output:")
	Done(out, 4200*time.Millisecond)
	VerboseStatus(out, "Upload complete", 2100*time.Millisecond)
	cmd.Println()

	// Verbose status
	cmd.Println("Verbose status (with timing):")
	VerboseStatus(out, "Prepared upload", 150*time.Millisecond)
	VerboseStatus(out, "Uploaded", 2100*time.Millisecond)
	VerboseStatus(out, "Finalized", 80*time.Millisecond)
	cmd.Println()

	return nil
}

func runAnimatedDemo(cmd *cobra.Command, out, stderr interface{ Write([]byte) (int, error) }, fast bool) error {
	delay := func(d time.Duration) {
		if !fast {
			time.Sleep(d)
		}
	}

	start := time.Now()

	cmd.Println("=== Upload Flow: Success ===")
	cmd.Println()

	Status(stderr, "Preparing upload for MyApp.zip…")
	delay(500 * time.Millisecond)
	Status(stderr, "Uploading to edge network…")
	delay(1500 * time.Millisecond)
	Status(stderr, "Finalizing upload…")
	delay(300 * time.Millisecond)
	Status(stderr, "Processing build…")
	delay(800 * time.Millisecond)
	Status(stderr, "Still processing…")
	delay(600 * time.Millisecond)

	cmd.Println()
	Successf(out, "Build %d processed", 42)
	Success(out, "Feed updated: https://app.usetwinkle.com/feeds/my-app/appcast.xml")
	Done(out, time.Since(start))

	// Failure flow
	cmd.Println()
	cmd.Println("=== Upload Flow: Failure ===")
	cmd.Println()

	failStart := time.Now()
	Status(stderr, "Preparing upload for MyApp.zip…")
	delay(500 * time.Millisecond)
	Status(stderr, "Uploading to edge network…")
	delay(1500 * time.Millisecond)
	Status(stderr, "Finalizing upload…")
	delay(300 * time.Millisecond)
	Status(stderr, "Processing build…")
	delay(1000 * time.Millisecond)

	cmd.Println()
	Errorf(out, "Build %d failed", 17)
	ErrorDetail(out, "version: Build number must be greater than 2022.02.03231334 for version 1.0")
	ErrorDetail(out, "signing: missing certificate")
	Done(out, time.Since(failStart))

	return nil
}
