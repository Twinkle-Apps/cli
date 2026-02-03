package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/twinkle-apps/cli/internal/api"
)

func TestPrintBuildResponseFailedSkipsAppcastWaitingMessage(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	resp := api.BuildResponse{
		Build: api.Build{
			ID:          11,
			Status:      "failed",
			InsertedAt:  api.APITime{Time: time.Now()},
			UpdatedAt:   api.APITime{Time: time.Now()},
			Version:     strPtr("1.0.0"),
			BuildNumber: strPtr("1"),
		},
		Appcast: api.Appcast{
			Status:  "waiting_manual",
			Message: "waiting on manual update in web portal",
			FeedURL: "https://example.com/feed.xml",
		},
	}

	printBuildResponse(cmd, resp, false)

	output := buf.String()
	if strings.Contains(output, "Awaiting manual publication") {
		t.Fatalf("expected failed build output to skip appcast waiting message, got %q", output)
	}
}

func TestPrintBuildResponseFailedShowsErrors(t *testing.T) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	resp := api.BuildResponse{
		Build: api.Build{
			ID:          11,
			Status:      "failed",
			InsertedAt:  api.APITime{Time: time.Now()},
			UpdatedAt:   api.APITime{Time: time.Now()},
			Version:     strPtr("1.0.0"),
			BuildNumber: strPtr("1"),
			Metadata: &api.BuildMetadata{
				ProcessingErrors: map[string]interface{}{
					"signing": []interface{}{"missing signature"},
					"bundle": map[string]interface{}{
						"message": "invalid bundle",
						"step":    "version",
					},
				},
			},
		},
		Appcast: api.Appcast{
			Status:  "waiting_manual",
			Message: "waiting on manual update in web portal",
			FeedURL: "https://example.com/feed.xml",
		},
	}

	printBuildResponse(cmd, resp, false)

	output := buf.String()
	if strings.Contains(output, "Appcast:") {
		t.Fatalf("expected appcast message to be omitted on failed output, got %q", output)
	}
	if !strings.Contains(output, "Errors:") || !strings.Contains(output, "version: invalid bundle") || !strings.Contains(output, "signing: missing signature") {
		t.Fatalf("expected processing errors in output, got %q", output)
	}
}

func strPtr(value string) *string {
	return &value
}
