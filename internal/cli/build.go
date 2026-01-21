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
		Use:     "build",
		Aliases: []string{"ship"},
		Short:   "Manage app builds",
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

			return renderOutput(cmd, appCtx.JSON, resp)
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

			fmt.Fprintf(cmd.ErrOrStderr(), "Waiting for build %s (timeout %ds)...\n", buildID, timeout)
			resp, err := pollBuildStatus(cmd.Context(), cmd.ErrOrStderr(), appCtx.Client, appID, buildID, "", timeout, pollInterval)
			if err != nil {
				return err
			}

			return renderOutput(cmd, appCtx.JSON, resp)
		},
	}

	cmd.Flags().IntVar(&timeout, "timeout", 0, "Wait timeout in seconds (max 300)")

	return cmd
}

func newBuildUploadCmd() *cobra.Command {
	var (
		wait    bool
		timeout int
	)
	const pollInterval = 5 * time.Second

	cmd := &cobra.Command{
		Use:   "upload <app-id> <file>",
		Short: "Upload a build",
		Args:  cobra.ExactArgs(2),
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

			resolvedContentType := "application/zip"
			params := api.BuildUploadParams{
				Version:     "0.0.0",
				BuildNumber: "0",
				ContentType: resolvedContentType,
			}

			createResp, err := appCtx.Client.CreateUpload(cmd.Context(), appID, params)
			if err != nil {
				return err
			}

			if err := appCtx.Client.UploadFile(cmd.Context(), createResp.UploadURL, filePath, resolvedContentType); err != nil {
				return err
			}

			buildID := createResp.BuildID.Int()
			completeResp, err := appCtx.Client.CompleteUpload(cmd.Context(), appID, buildID)
			if err != nil {
				return err
			}

			if !wait {
				return renderOutput(cmd, appCtx.JSON, completeResp)
			}

			fmt.Fprintf(cmd.ErrOrStderr(), "Waiting for build %d (timeout %ds)...\n", buildID, timeout)
			waitResp, err := pollBuildStatus(cmd.Context(), cmd.ErrOrStderr(), appCtx.Client, appID, fmt.Sprintf("%d", buildID), completeResp.StatusURL, timeout, pollInterval)
			if err != nil {
				return err
			}
			return renderOutput(cmd, appCtx.JSON, waitResp)
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", false, "Wait for processing to complete")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Wait timeout in seconds (max 300)")

	_ = cmd.MarkFlagFilename("file")

	return cmd
}

func pollBuildStatus(ctx context.Context, stderr io.Writer, client *api.Client, appID, buildID, statusURL string, timeoutSeconds int, interval time.Duration) (api.BuildResponse, error) {
	deadline := time.Time{}
	if timeoutSeconds > 0 {
		deadline = time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	}

	for {
		var (
			resp api.BuildResponse
			err  error
		)
		if statusURL != "" {
			resp, err = client.GetBuildByURL(ctx, statusURL)
		} else {
			resp, err = client.GetBuild(ctx, appID, buildID)
		}
		if err != nil {
			return api.BuildResponse{}, err
		}

		if resp.Build.Status != "processing" {
			return resp, nil
		}

		fmt.Fprintln(stderr, "Still waiting... status=processing")

		if !deadline.IsZero() && time.Now().After(deadline) {
			return api.BuildResponse{}, fmt.Errorf("wait timeout after %ds", timeoutSeconds)
		}

		select {
		case <-ctx.Done():
			return api.BuildResponse{}, ctx.Err()
		case <-time.After(interval):
		}
	}
}
