package cli

import (
	"bytes"
	"encoding/json"
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
	if !strings.Contains(output, "version: invalid bundle") || !strings.Contains(output, "signing: missing signature") {
		t.Fatalf("expected processing errors in output, got %q", output)
	}
	if !strings.Contains(output, "✕") {
		t.Fatalf("expected error symbol in output, got %q", output)
	}
	if !strings.Contains(output, "↳") {
		t.Fatalf("expected error detail connector in output, got %q", output)
	}
}

func strPtr(value string) *string {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func newTestCmd() (*cobra.Command, *bytes.Buffer) {
	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	return cmd, &buf
}

// --- JSON output tests ---

func TestRenderOutputJSONBuildResponseAvailable(t *testing.T) {
	cmd, buf := newTestCmd()

	pubTime := api.APITime{Time: time.Date(2026, 1, 15, 10, 30, 0, 0, time.UTC)}
	feedURL := "https://example.com/appcast.xml"

	resp := api.BuildResponse{
		Build: api.Build{
			ID:          42,
			Status:      "available",
			Version:     strPtr("1.2.0"),
			BuildNumber: strPtr("5"),
			InsertedAt:  api.APITime{Time: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)},
			UpdatedAt:   api.APITime{Time: time.Date(2026, 1, 15, 10, 29, 0, 0, time.UTC)},
			Metadata: &api.BuildMetadata{
				BuildVersion: strPtr("1.2.0"),
				BuildNumber:  strPtr("5"),
				BuildSize:    intPtr(1048576),
				Signature:    strPtr("abc123"),
			},
		},
		Appcast: api.Appcast{
			Status:      "published",
			FeedURL:     feedURL,
			Message:     "published",
			PublishedAt: &pubTime,
			URL:         &feedURL,
		},
		PollAfterMs: intPtr(5000),
	}

	if err := renderOutput(cmd, true, false, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got api.BuildResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}

	if got.Build.ID != 42 {
		t.Errorf("build ID: got %d, want 42", got.Build.ID)
	}
	if got.Build.Status != "available" {
		t.Errorf("status: got %q, want %q", got.Build.Status, "available")
	}
	if got.Build.Version == nil || *got.Build.Version != "1.2.0" {
		t.Errorf("version: got %v, want \"1.2.0\"", got.Build.Version)
	}
	if got.Appcast.Status != "published" {
		t.Errorf("appcast status: got %q, want %q", got.Appcast.Status, "published")
	}
	if got.Appcast.FeedURL != feedURL {
		t.Errorf("feed URL: got %q, want %q", got.Appcast.FeedURL, feedURL)
	}
	if got.Build.Metadata == nil {
		t.Fatal("metadata: got nil, want populated")
	}
	if got.Build.Metadata.BuildSize == nil || *got.Build.Metadata.BuildSize != 1048576 {
		t.Errorf("build size: got %v, want 1048576", got.Build.Metadata.BuildSize)
	}
	if got.PollAfterMs == nil || *got.PollAfterMs != 5000 {
		t.Errorf("poll_after_ms: got %v, want 5000", got.PollAfterMs)
	}
}

func TestRenderOutputJSONBuildResponseFailed(t *testing.T) {
	cmd, buf := newTestCmd()

	resp := api.BuildResponse{
		Build: api.Build{
			ID:         11,
			Status:     "failed",
			InsertedAt: api.APITime{Time: time.Now()},
			UpdatedAt:  api.APITime{Time: time.Now()},
			Metadata: &api.BuildMetadata{
				ProcessingErrors: map[string]interface{}{
					"signing": "missing certificate",
					"version": "build number too low",
				},
			},
		},
		Appcast: api.Appcast{
			Status:  "waiting_manual",
			FeedURL: "https://example.com/feed.xml",
		},
	}

	if err := renderOutput(cmd, true, false, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got api.BuildResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}

	if got.Build.Status != "failed" {
		t.Errorf("status: got %q, want %q", got.Build.Status, "failed")
	}
	if got.Build.Metadata == nil || len(got.Build.Metadata.ProcessingErrors) != 2 {
		t.Fatalf("processing errors: got %v, want 2 entries", got.Build.Metadata)
	}
	if got.Build.Metadata.ProcessingErrors["signing"] != "missing certificate" {
		t.Errorf("signing error: got %v, want \"missing certificate\"", got.Build.Metadata.ProcessingErrors["signing"])
	}
	if got.Build.Metadata.ProcessingErrors["version"] != "build number too low" {
		t.Errorf("version error: got %v, want \"build number too low\"", got.Build.Metadata.ProcessingErrors["version"])
	}
}

// PollAfterMs has omitempty — nil means the key is absent entirely.
// Other pointer fields (Metadata, Version) have no omitempty — nil serializes as null.
func TestRenderOutputJSONBuildResponseNilFields(t *testing.T) {
	cmd, buf := newTestCmd()

	resp := api.BuildResponse{
		Build: api.Build{
			ID:         1,
			Status:     "processing",
			InsertedAt: api.APITime{Time: time.Now()},
			UpdatedAt:  api.APITime{Time: time.Now()},
			// Version, BuildNumber, Metadata all nil
		},
		Appcast: api.Appcast{Status: "waiting"},
		// PollAfterMs nil
	}

	if err := renderOutput(cmd, true, false, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	if strings.Contains(output, "poll_after_ms") {
		t.Errorf("poll_after_ms should be omitted when nil, got: %s", output)
	}
	if !strings.Contains(output, `"metadata": null`) {
		t.Errorf("metadata should serialize as null, got: %s", output)
	}
	if !strings.Contains(output, `"version": null`) {
		t.Errorf("version should serialize as null, got: %s", output)
	}
}

func TestRenderOutputJSONUploadComplete(t *testing.T) {
	cmd, buf := newTestCmd()

	// BuildID.value is unexported; populate via unmarshal of a known fixture.
	fixture := `{"build_id":99,"status_url":"https://example.com/status","upload_state":"complete","wait_url":"https://example.com/wait"}`
	var resp api.BuildUploadCompleteResponse
	if err := json.Unmarshal([]byte(fixture), &resp); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := renderOutput(cmd, true, false, resp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var got api.BuildUploadCompleteResponse
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, buf.String())
	}

	if got.BuildID.Int() != 99 {
		t.Errorf("build ID: got %d, want 99", got.BuildID.Int())
	}
	if got.StatusURL != "https://example.com/status" {
		t.Errorf("status URL: got %q, want %q", got.StatusURL, "https://example.com/status")
	}
	if got.WaitURL != "https://example.com/wait" {
		t.Errorf("wait URL: got %q, want %q", got.WaitURL, "https://example.com/wait")
	}
	// BuildID must be a bare integer, not a quoted string.
	if strings.Contains(buf.String(), `"build_id": "`) {
		t.Errorf("build_id should be an integer, got string in: %s", buf.String())
	}
}

// Regression: none of our styled-output symbols or ANSI escape sequences
// should appear when the JSON path is taken.
func TestRenderOutputJSONContainsNoStyling(t *testing.T) {
	styleChars := []string{"\x1b[", "·", "✓", "✕", "↳"}

	t.Run("BuildResponse", func(t *testing.T) {
		cmd, buf := newTestCmd()
		resp := api.BuildResponse{
			Build: api.Build{
				ID:         5,
				Status:     "failed",
				InsertedAt: api.APITime{Time: time.Now()},
				UpdatedAt:  api.APITime{Time: time.Now()},
				Metadata: &api.BuildMetadata{
					ProcessingErrors: map[string]interface{}{
						"version": "too low",
					},
				},
			},
			Appcast: api.Appcast{Status: "waiting_manual"},
		}
		if err := renderOutput(cmd, true, false, resp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, s := range styleChars {
			if strings.Contains(buf.String(), s) {
				t.Errorf("found style artifact %q in JSON output: %s", s, buf.String())
			}
		}
	})

	t.Run("UploadComplete", func(t *testing.T) {
		cmd, buf := newTestCmd()
		fixture := `{"build_id":1,"status_url":"https://example.com/s","upload_state":"complete","wait_url":"https://example.com/w"}`
		var resp api.BuildUploadCompleteResponse
		if err := json.Unmarshal([]byte(fixture), &resp); err != nil {
			t.Fatalf("setup: %v", err)
		}
		if err := renderOutput(cmd, true, false, resp); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, s := range styleChars {
			if strings.Contains(buf.String(), s) {
				t.Errorf("found style artifact %q in JSON output: %s", s, buf.String())
			}
		}
	})
}

func TestRenderOutputUnsupportedType(t *testing.T) {
	cmd, _ := newTestCmd()
	err := renderOutput(cmd, false, false, "not a valid payload")
	if err == nil {
		t.Fatal("expected error for unsupported type, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output type") {
		t.Errorf("expected unsupported output type error, got: %v", err)
	}
}
