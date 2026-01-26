//go:build !debug

package cli

import "github.com/spf13/cobra"

func init() {
	// No-op for release builds - demo command is not available
	registerDemoCommand = func(*cobra.Command) {}
}
