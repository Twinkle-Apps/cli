package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
)

const defaultTimeout = 30 * time.Second
const defaultWaitTimeout = defaultTimeout + 10*time.Second

var ErrMissingAPIKey = errors.New("missing API key")

// Client wraps Twinkle API calls.
type Client struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string, httpClient *http.Client) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, ErrMissingAPIKey
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse base url: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}
	return &Client{baseURL: parsed, apiKey: apiKey, httpClient: httpClient}, nil
}

func (c *Client) GetBuild(ctx context.Context, appID, buildID string) (BuildResponse, error) {
	endpoint := c.withPath("/api/v1/apps/%s/builds/%s", appID, buildID)
	var resp BuildResponse
	if err := c.doJSON(ctx, http.MethodGet, endpoint, nil, &resp); err != nil {
		return BuildResponse{}, err
	}
	return resp, nil
}

func (c *Client) GetBuildByURL(ctx context.Context, statusURL string) (BuildResponse, error) {
	if strings.TrimSpace(statusURL) == "" {
		return BuildResponse{}, fmt.Errorf("status url is empty")
	}
	parsed, err := url.Parse(statusURL)
	if err != nil {
		return BuildResponse{}, fmt.Errorf("parse status url: %w", err)
	}
	if parsed.Scheme == "" {
		parsed = c.baseURL.ResolveReference(parsed)
	}
	var resp BuildResponse
	if err := c.doJSON(ctx, http.MethodGet, parsed, nil, &resp); err != nil {
		return BuildResponse{}, err
	}
	return resp, nil
}

func (c *Client) WaitBuild(ctx context.Context, appID, buildID string, timeoutSeconds int) (BuildResponse, error) {
	endpoint := c.withPath("/api/v1/apps/%s/builds/%s/wait", appID, buildID)
	if timeoutSeconds > 0 {
		query := endpoint.Query()
		query.Set("timeout", fmt.Sprintf("%d", timeoutSeconds))
		endpoint.RawQuery = query.Encode()
	}
	var resp BuildResponse
	client := c.waitClient(timeoutSeconds)
	if err := c.doJSONWithClient(ctx, client, http.MethodGet, endpoint, nil, &resp); err != nil {
		return BuildResponse{}, err
	}
	return resp, nil
}

func (c *Client) WaitBuildByURL(ctx context.Context, waitURL string, timeoutSeconds int) (BuildResponse, error) {
	if strings.TrimSpace(waitURL) == "" {
		return BuildResponse{}, fmt.Errorf("wait url is empty")
	}
	parsed, err := url.Parse(waitURL)
	if err != nil {
		return BuildResponse{}, fmt.Errorf("parse wait url: %w", err)
	}
	if parsed.Scheme == "" {
		parsed = c.baseURL.ResolveReference(parsed)
	}
	if timeoutSeconds > 0 {
		query := parsed.Query()
		query.Set("timeout", fmt.Sprintf("%d", timeoutSeconds))
		parsed.RawQuery = query.Encode()
	}
	var resp BuildResponse
	client := c.waitClient(timeoutSeconds)
	if err := c.doJSONWithClient(ctx, client, http.MethodGet, parsed, nil, &resp); err != nil {
		return BuildResponse{}, err
	}
	return resp, nil
}

func (c *Client) CreateUpload(ctx context.Context, appID string, params BuildUploadParams) (BuildUploadResponse, error) {
	return c.CreateUploadWithOptions(ctx, appID, params)
}

type CreateUploadOption func(*createUploadOptions)

type createUploadOptions struct {
	idempotencyKey string
}

func WithIdempotencyKey(key string) CreateUploadOption {
	return func(opts *createUploadOptions) {
		opts.idempotencyKey = key
	}
}

func (c *Client) CreateUploadWithOptions(ctx context.Context, appID string, params BuildUploadParams, opts ...CreateUploadOption) (BuildUploadResponse, error) {
	endpoint := c.withPath("/api/v1/apps/%s/uploads", appID)
	body := BuildUploadRequest{Build: params}
	var resp BuildUploadResponse
	options := createUploadOptions{}
	for _, opt := range opts {
		opt(&options)
	}
	headers := map[string]string{}
	idempotencyKey := strings.TrimSpace(options.idempotencyKey)
	if idempotencyKey == "" {
		idempotencyKey = uuid.NewString()
	}
	headers["Idempotency-Key"] = idempotencyKey
	if err := c.doJSONWithHeaders(ctx, http.MethodPost, endpoint, body, &resp, headers); err != nil {
		return BuildUploadResponse{}, err
	}
	return resp, nil
}

func (c *Client) CompleteUpload(ctx context.Context, appID string, buildID int) (BuildUploadCompleteResponse, error) {
	endpoint := c.withPath("/api/v1/apps/%s/uploads/%d/complete", appID, buildID)
	var resp BuildUploadCompleteResponse
	if err := c.doJSON(ctx, http.MethodPost, endpoint, nil, &resp); err != nil {
		return BuildUploadCompleteResponse{}, err
	}
	return resp, nil
}

func (c *Client) UploadFile(ctx context.Context, uploadURL, filePath, contentType string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, uploadURL, file)
	if err != nil {
		return fmt.Errorf("create upload request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.ContentLength = stat.Size()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<10))
		return fmt.Errorf("upload file: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (c *Client) withPath(format string, args ...interface{}) *url.URL {
	rel := fmt.Sprintf(format, args...)
	urlCopy := *c.baseURL
	urlCopy.Path = path.Join(strings.TrimSuffix(c.baseURL.Path, "/"), rel)
	return &urlCopy
}

func (c *Client) doJSON(ctx context.Context, method string, endpoint *url.URL, body interface{}, target interface{}) error {
	return c.doJSONWithClient(ctx, c.httpClient, method, endpoint, body, target)
}

func (c *Client) doJSONWithClient(ctx context.Context, client *http.Client, method string, endpoint *url.URL, body interface{}, target interface{}) error {
	return c.doJSONWithHeadersAndClient(ctx, client, method, endpoint, body, target, nil)
}

func (c *Client) doJSONWithHeaders(ctx context.Context, method string, endpoint *url.URL, body interface{}, target interface{}, headers map[string]string) error {
	return c.doJSONWithHeadersAndClient(ctx, c.httpClient, method, endpoint, body, target, headers)
}

func (c *Client) doJSONWithHeadersAndClient(ctx context.Context, client *http.Client, method string, endpoint *url.URL, body interface{}, target interface{}, headers map[string]string) error {
	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return decodeAPIError(resp.Body, resp.StatusCode)
	}

	if target == nil {
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func (c *Client) waitClient(timeoutSeconds int) *http.Client {
	custom := *c.httpClient
	if timeoutSeconds > 0 {
		custom.Timeout = time.Duration(timeoutSeconds+10) * time.Second
	} else {
		// Ensure long-poll waits for the server default timeout plus buffer.
		if custom.Timeout <= 0 || custom.Timeout < defaultWaitTimeout {
			custom.Timeout = defaultWaitTimeout
		}
	}
	return &custom
}

func decodeAPIError(body io.Reader, status int) error {
	payload, err := io.ReadAll(io.LimitReader(body, 32<<10))
	if err != nil {
		return fmt.Errorf("api error status %d", status)
	}
	var apiErr ErrorResponse
	if jsonErr := json.Unmarshal(payload, &apiErr); jsonErr == nil && apiErr.Error != "" {
		if len(apiErr.Details) > 0 {
			if detailPayload, err := json.Marshal(apiErr.Details); err == nil {
				return fmt.Errorf("api error status %d: %s: %s", status, apiErr.Error, strings.TrimSpace(string(detailPayload)))
			}
		}
		return fmt.Errorf("api error status %d: %s", status, apiErr.Error)
	}
	return fmt.Errorf("api error status %d: %s", status, strings.TrimSpace(string(payload)))
}
