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
	Status(out, "Preparing upload...")
	Status(out, "Uploading MyApp.zip...")
	Statusf(out, "Processing build %d...", 42)
	cmd.Println()

	// Success messages (green checkmark)
	cmd.Println("Success messages (green checkmark):")
	Success(out, "Build uploaded successfully")
	Successf(out, "Published to %d regions", 12)
	Success(out, "Feed updated: https://example.com/appcast.xml")
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

	cmd.Println("=== Simulated Upload Flow ===")
	cmd.Println()

	// Step 1: Prepare
	Status(stderr, "Preparing upload for MyApp.zip...")
	delay(500 * time.Millisecond)

	// Step 2: Upload
	Status(stderr, "Uploading MyApp.zip...")
	delay(1500 * time.Millisecond)

	// Step 3: Finalize
	Status(stderr, "Finalizing upload...")
	delay(300 * time.Millisecond)

	// Step 4: Process
	Status(stderr, "Processing build...")
	delay(800 * time.Millisecond)
	Status(stderr, "Still processing...")
	delay(600 * time.Millisecond)

	// Success output
	cmd.Println()
	Successf(out, "Build %d processed", 42)
	Successf(out, "Published to %d regions", 12)
	Success(out, "Feed updated: https://app.usetwinkle.com/feeds/my-app/appcast.xml")
	cmd.Println()

	Done(stderr, time.Since(start))

	return nil
}
