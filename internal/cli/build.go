package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/twinkle-apps/cli/internal/api"
)

func newBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "build",
		Short: "Manage app builds",
	}

	cmd.AddCommand(newBuildStatusCmd())
	cmd.AddCommand(newBuildWaitCmd())
	cmd.AddCommand(newBuildUploadCmd())

	return cmd
}

func newBuildStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <app-id> <build-id>",
		Short: "Get build status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID := args[0]
			buildID := args[1]

			appCtx, err := getAppContext(cmd)
			if err != nil {
				return err
			}

			resp, err := appCtx.Client.GetBuild(cmd.Context(), appID, buildID)
			if err != nil {
				return err
			}

			return renderOutput(cmd, appCtx.JSON, appCtx.Verbose, resp)
		},
	}

	return cmd
}

func newBuildWaitCmd() *cobra.Command {
	var timeout int
	const pollInterval = 5 * time.Second

	cmd := &cobra.Command{
		Use:   "wait <app-id> <build-id>",
		Short: "Wait for build processing",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID := args[0]
			buildID := args[1]

			if timeout < 0 {
				return errors.New("timeout must be >= 0")
			}
			if timeout > 300 {
				return errors.New("timeout must be <= 300")
			}

			appCtx, err := getAppContext(cmd)
			if err != nil {
				return err
			}

			stderr := cmd.ErrOrStderr()
			start := time.Now()
			jsonOut := appCtx.JSON

			if !jsonOut {
				Statusf(stderr, "Waiting for build %s...", buildID)
			}
			resp, err := pollBuildStatus(cmd.Context(), stderr, appCtx.Client, appID, buildID, "", timeout, pollInterval, appCtx.Verbose, jsonOut)
			if err != nil {
				return err
			}

			if err := renderOutput(cmd, jsonOut, appCtx.Verbose, resp); err != nil {
				return err
			}
			if !jsonOut {
				Done(stderr, time.Since(start))
			}
			return nil
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 0, "Wait timeout in seconds (max 300)")

	return cmd
}

func newBuildUploadCmd() *cobra.Command {
	return newBuildUploadCmdWithUse("upload <app-id> <file>", "Upload a build", nil)
}

func newShipCmd() *cobra.Command {
	return newBuildUploadCmdWithUse("ship <app-id> <file>", "Alias for build upload", nil)
}

func newBuildUploadCmdWithUse(use, short string, aliases []string) *cobra.Command {
	var (
		wait    bool
		timeout int
	)
	const pollInterval = 5 * time.Second

	cmd := &cobra.Command{
		Use:     use,
		Short:   short,
		Aliases: aliases,
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			appID := args[0]
			filePath := args[1]

			if strings.TrimSpace(filePath) == "" {
				return errors.New("file path is required")
			}
			if _, err := os.Stat(filePath); err != nil {
				return fmt.Errorf("file not accessible: %w", err)
			}
			if strings.ToLower(filepath.Ext(filePath)) != ".zip" {
				return errors.New("only .zip archives are supported")
			}
			if timeout < 0 {
				return errors.New("timeout must be >= 0")
			}
			if timeout > 300 {
				return errors.New("timeout must be <= 300")
			}

			appCtx, err := getAppContext(cmd)
			if err != nil {
				return err
			}

			stderr := cmd.ErrOrStderr()
			totalStart := time.Now()
			verbose := appCtx.Verbose
			jsonOut := appCtx.JSON

			// Step 1: Prepare upload
			stepStart := time.Now()
			if !jsonOut {
				Statusf(stderr, "Preparing upload for %s...", filepath.Base(filePath))
			}

			resolvedContentType := "application/zip"
			params := api.BuildUploadParams{
				ContentType: resolvedContentType,
			}

			createResp, err := appCtx.Client.CreateUpload(cmd.Context(), appID, params)
			if err != nil {
				return err
			}
			if verbose && !jsonOut {
				VerboseStatus(stderr, "Prepared upload", time.Since(stepStart))
			}

			// Step 2: Upload file
			stepStart = time.Now()
			if !jsonOut {
				Statusf(stderr, "Uploading to edge network…")
			}

			if err := appCtx.Client.UploadFile(cmd.Context(), createResp.UploadURL, filePath, resolvedContentType); err != nil {
				return err
			}
			if verbose && !jsonOut {
				VerboseStatus(stderr, "Uploaded", time.Since(stepStart))
			}

			// Step 3: Complete upload
			stepStart = time.Now()
			if !jsonOut {
				Status(stderr, "Finalizing upload…")
			}

			buildID := createResp.BuildID.Int()
			completeResp, err := appCtx.Client.CompleteUpload(cmd.Context(), appID, buildID)
			if err != nil {
				return err
			}
			if verbose && !jsonOut {
				VerboseStatus(stderr, "Finalized", time.Since(stepStart))
			}

			if !wait {
				if err := renderOutput(cmd, jsonOut, verbose, completeResp); err != nil {
					return err
				}
				if !jsonOut {
					Done(stderr, time.Since(totalStart))
				}
				return nil
			}

			// Step 4: Wait for processing
			stepStart = time.Now()
			if !jsonOut {
				Status(stderr, "Processing build…")
			}

			waitResp, err := pollBuildStatus(cmd.Context(), stderr, appCtx.Client, appID, fmt.Sprintf("%d", buildID), completeResp.WaitURL, timeout, pollInterval, verbose, jsonOut)
			if err != nil {
				return err
			}
			if verbose && !jsonOut {
				VerboseStatus(stderr, "Processing complete", time.Since(stepStart))
			}

			if err := renderOutput(cmd, jsonOut, verbose, waitResp); err != nil {
				return err
			}
			if !jsonOut {
				Done(stderr, time.Since(totalStart))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for processing to complete")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Wait timeout in seconds (max 300)")

	_ = cmd.MarkFlagFilename("file")

	return cmd
}

func pollBuildStatus(ctx context.Context, stderr io.Writer, client *api.Client, appID, buildID, waitURL string, timeoutSeconds int, interval time.Duration, verbose, jsonOut bool) (api.BuildResponse, error) {
	deadline := time.Time{}
	if timeoutSeconds > 0 {
		deadline = time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	}

	pollStart := time.Now()

	for {
		var (
			resp api.BuildResponse
			err  error
		)
		if waitURL != "" {
			resp, err = client.WaitBuildByURL(ctx, waitURL, timeoutSeconds)
		} else {
			resp, err = client.WaitBuild(ctx, appID, buildID, timeoutSeconds)
		}
		if err != nil {
			return api.BuildResponse{}, err
		}

		if resp.Build.Status != "processing" {
			return resp, nil
		}

		if !jsonOut {
			if verbose {
				VerboseStatus(stderr, "Still processing…", time.Since(pollStart))
			} else {
				Status(stderr, "Still processing…")
			}
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return resp, nil
		}

		// Respect server-guided backoff when the wait endpoint returns 202.
		nextInterval := interval
		if resp.PollAfterMs != nil && *resp.PollAfterMs > 0 {
			nextInterval = time.Duration(*resp.PollAfterMs) * time.Millisecond
		}
		if !deadline.IsZero() {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				return resp, nil
			}
			if nextInterval > remaining {
				nextInterval = remaining
			}
		}

		select {
		case <-ctx.Done():
			return api.BuildResponse{}, ctx.Err()
		case <-time.After(nextInterval):
		}
	}
}
