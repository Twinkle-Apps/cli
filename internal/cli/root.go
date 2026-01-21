package cli

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/twinkle-apps/cli/internal/api"
)

const (
	defaultBaseURL = "https://app.usetwinkle.com"
	envAPIKey      = "TWINKLE_API_KEY"
	envBaseURL     = "TWINKLE_BASE_URL"
)

type appContextKey struct{}

type AppContext struct {
	Client *api.Client
	JSON   bool
}

func Execute() error {
	root := newRootCmd()
	return root.Execute()
}

func newRootCmd() *cobra.Command {
	var (
		apiKey  string
		baseURL string
		jsonOut bool
	)

	cmd := &cobra.Command{
		Use:   "twinkle",
		Short: "Twinkle CLI",
		Long:  "Command-line interface for the Twinkle build API.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Name() == "version" {
				return nil
			}

			if apiKey == "" {
				apiKey = os.Getenv(envAPIKey)
			}
			if baseURL == "" {
				baseURL = os.Getenv(envBaseURL)
				if baseURL == "" {
					baseURL = defaultBaseURL
				}
			}

			client, err := api.NewClient(baseURL, apiKey, nil)
			if err != nil {
				if errors.Is(err, api.ErrMissingAPIKey) {
					return fmt.Errorf("api key is required: set --api-key or %s", envAPIKey)
				}
				return err
			}

			ctx := context.WithValue(cmd.Context(), appContextKey{}, &AppContext{Client: client, JSON: jsonOut})
			cmd.SetContext(ctx)
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "Twinkle API key (overrides "+envAPIKey+")")
	cmd.PersistentFlags().StringVar(&baseURL, "base-url", "", "Twinkle API base URL (overrides "+envBaseURL+")")
	cmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output JSON")

	cmd.AddCommand(newBuildCmd())
	cmd.AddCommand(newVersionCmd())

	return cmd
}

func getAppContext(cmd *cobra.Command) (*AppContext, error) {
	ctx := cmd.Context().Value(appContextKey{})
	if ctx == nil {
		return nil, errors.New("missing app context")
	}
	appCtx, ok := ctx.(*AppContext)
	if !ok {
		return nil, errors.New("invalid app context")
	}
	return appCtx, nil
}
