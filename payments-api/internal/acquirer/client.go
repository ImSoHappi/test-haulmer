package acquirer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/domain"
)

// ErrAuthorizationNotFound is returned when on-demand reconciliation confirms no transaction exists.
var ErrAuthorizationNotFound = errors.New("authorization not found in acquirer")

// ClientConfig holds configuration parameters for the resilient acquirer HTTP client.
type ClientConfig struct {
	BaseURL          string
	RequestTimeout   time.Duration
	ReconcileTimeout time.Duration
	InitialBackoff   time.Duration
	MaxRetries       int
	HTTPClient       *http.Client
}

// Client implements AcquirerPort with exponential backoff, jitter, and on-demand reconciliation.
type Client struct {
	config ClientConfig
	client *http.Client
}

// NewClient constructs a new resilient acquirer client.
// Returns an error if BaseURL is empty (fail-fast, eliminating magic defaults).
func NewClient(cfg ClientConfig) (*Client, error) {
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	if cfg.BaseURL == "" {
		return nil, errors.New("acquirer: BaseURL is required")
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = 2 * time.Second
	}
	if cfg.ReconcileTimeout <= 0 {
		cfg.ReconcileTimeout = 1 * time.Second
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = 200 * time.Millisecond
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 2
	}

	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}

	return &Client{
		config: cfg,
		client: httpClient,
	}, nil
}

// Authorize attempts to authorize a payment through the acquirer.
// It handles transient 503 retries and on-demand reconciliation on timeout.
// Business decisions (APPROVED, DECLINED) and 4xx client errors are never retried.
func (c *Client) Authorize(ctx context.Context, req AuthorizeRequest) (*AcquirerResponse, error) {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal acquirer authorize payload: %w", err)
	}

	url := c.config.BaseURL + "/authorizations"

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			if err := c.waitForRetry(ctx, attempt); err != nil {
				return nil, err
			}
		}

		resp, isTimeout, err := c.doPost(ctx, url, bodyBytes, req.CorrelationID)
		if isTimeout {
			return c.reconcileTimeout(ctx, req.ReferenceID, req.CorrelationID)
		}
		if err != nil {
			if attempt < c.config.MaxRetries {
				continue
			}
			return nil, domain.ErrAcquirerUnavailable
		}

		authResp, retry, err := c.parseAuthorizeResponse(resp)
		if retry {
			if attempt < c.config.MaxRetries {
				continue
			}
			return nil, domain.ErrAcquirerUnavailable
		}
		return authResp, err
	}

	return nil, domain.ErrAcquirerUnavailable
}

func (c *Client) waitForRetry(ctx context.Context, attempt int) error {
	backoff := c.calculateBackoff(attempt)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(backoff):
		return nil
	}
}

func (c *Client) parseAuthorizeResponse(resp *http.Response) (*AcquirerResponse, bool, error) {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable {
		return nil, true, nil
	}

	if resp.StatusCode == http.StatusOK {
		var authResp AcquirerResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return nil, false, domain.ErrAcquirerUnexpected
		}
		return &authResp, false, nil
	}

	return nil, false, domain.ErrAcquirerUnexpected
}

// GetAuthorization reconciles the status of a transaction with the acquirer.
func (c *Client) GetAuthorization(ctx context.Context, referenceID string, correlationID string) (*AcquirerResponse, error) {
	url := fmt.Sprintf("%s/authorizations/%s", c.config.BaseURL, referenceID)

	reqCtx, cancel := context.WithTimeout(ctx, c.config.ReconcileTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create reconcile request: %w", err)
	}

	if correlationID != "" {
		httpReq.Header.Set("X-Correlation-ID", correlationID)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrAuthorizationNotFound
	}

	if resp.StatusCode == http.StatusOK {
		var authResp AcquirerResponse
		if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
			return nil, domain.ErrAcquirerUnexpected
		}
		return &authResp, nil
	}

	return nil, fmt.Errorf("acquirer reconciliation returned status %d", resp.StatusCode)
}

func (c *Client) doPost(parentCtx context.Context, url string, body []byte, correlationID string) (*http.Response, bool, error) {
	reqCtx, cancel := context.WithTimeout(parentCtx, c.config.RequestTimeout)
	defer cancel()

	httpReq, err := http.NewRequestWithContext(reqCtx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, false, fmt.Errorf("failed to build http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if correlationID != "" {
		httpReq.Header.Set("X-Correlation-ID", correlationID)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		if errors.Is(reqCtx.Err(), context.DeadlineExceeded) || isNetTimeout(err) {
			return nil, true, err
		}
		return nil, false, err
	}

	return resp, false, nil
}

func (c *Client) reconcileTimeout(ctx context.Context, referenceID, correlationID string) (*AcquirerResponse, error) {
	reconResp, err := c.GetAuthorization(ctx, referenceID, correlationID)
	if err == nil && reconResp != nil && reconResp.Status == "APPROVED" {
		// Acquirer had processed the payment before the timeout
		return reconResp, nil
	}

	if errors.Is(err, ErrAuthorizationNotFound) {
		// Acquirer confirms no transaction was recorded; safe to mark as FAILED
		return &AcquirerResponse{
			ReferenceID: referenceID,
			Status:      "FAILED",
			Code:        "gateway_timeout",
			CreatedAt:   time.Now(),
		}, nil
	}

	// Acquirer reconciliation also failed or timed out; outcome unknown
	return nil, domain.ErrAcquirerUnknownResult
}

func (c *Client) calculateBackoff(attempt int) time.Duration {
	factor := 1 << (attempt - 1)
	base := c.config.InitialBackoff * time.Duration(factor)

	// Jitter: +/- 20%
	jitterRange := int64(base / 5)
	if jitterRange <= 0 {
		return base
	}
	jitter := time.Duration(rand.Int63n(jitterRange*2) - jitterRange)
	res := base + jitter
	if res < 0 {
		return base
	}
	return res
}

func isNetTimeout(err error) bool {
	if err == nil {
		return false
	}
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return strings.Contains(strings.ToLower(err.Error()), "timeout") ||
		strings.Contains(strings.ToLower(err.Error()), "deadline exceeded")
}

// DrainAndCloseResponseBody helper
func DrainAndCloseResponseBody(body io.ReadCloser) {
	if body != nil {
		_, _ = io.Copy(io.Discard, body)
		_ = body.Close()
	}
}
