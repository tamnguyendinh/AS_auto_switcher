package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"antigravity-switch/internal/connector"
)

const (
	// DefaultGoogleQuotaURL is the authoritative Google Cloud Code quota endpoint.
	DefaultGoogleQuotaURL = "https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota"

	// Standard health statuses for accounts.
	StatusPristine  = "Pristine"
	StatusPlentiful = "Plentiful"
	StatusLowGemini = "Low Gemini"
	StatusLowClaude = "Low Claude"
	StatusLowGPT    = "Low GPT"
	StatusLow       = "Low"
	StatusDepleted  = "Depleted"
	StatusError     = "Error"
)

var (
	// ErrEmptyRefreshToken is returned when the refresh token passed is empty.
	ErrEmptyRefreshToken = errors.New("refresh token is empty")

	// ErrQuotaFetchFailed is returned when retrieving quota from Google fails.
	ErrQuotaFetchFailed = errors.New("failed to retrieve user quota from google")

	// ErrNilRefresher is returned when the OAuth refresher is nil.
	ErrNilRefresher = errors.New("oauth refresher cannot be nil")
)

// ModelQuotaBucket represents quota status for an individual model or aggregated model family.
type ModelQuotaBucket struct {
	ModelID           string    `json:"model_id"`
	RemainingFraction float64   `json:"remaining_fraction"` // 0.0 to 1.0 (e.g. 0.235 = 23.5%)
	RemainingAmount   int64     `json:"remaining_amount,omitempty"`
	ResetTime         time.Time `json:"reset_time"`
	TokenType         string    `json:"token_type,omitempty"`
}

// AccountQuotaData encapsulates the parsed quota telemetry across model families for one account.
type AccountQuotaData struct {
	Email        string             `json:"email"`
	Claude       ModelQuotaBucket   `json:"claude"`
	Gemini       ModelQuotaBucket   `json:"gemini"`
	GPT          ModelQuotaBucket   `json:"gpt"`
	RawBuckets   []ModelQuotaBucket `json:"raw_buckets,omitempty"`
	ResetTime    time.Time          `json:"reset_time"` // Canonical reset time (UTC)
	Status       string             `json:"status"`     // Pristine, Plentiful, Low Gemini, Depleted, Error
	ErrorMessage string             `json:"error_message,omitempty"`
	LastUpdated  time.Time          `json:"last_updated"`
}

// QuotaClient communicates with Google Cloud Code internal quota endpoints.
type QuotaClient struct {
	refresher  connector.OAuthRefresher
	endpoint   string
	httpClient *http.Client
}

// NewQuotaClient creates a QuotaClient with default Google Cloud Code endpoint.
func NewQuotaClient(refresher connector.OAuthRefresher, httpClient *http.Client) *QuotaClient {
	return NewQuotaClientWithEndpoint(refresher, DefaultGoogleQuotaURL, httpClient)
}

// NewQuotaClientWithEndpoint creates a QuotaClient with custom endpoint and HTTP client (ideal for tests).
func NewQuotaClientWithEndpoint(refresher connector.OAuthRefresher, endpoint string, httpClient *http.Client) *QuotaClient {
	if endpoint == "" {
		endpoint = DefaultGoogleQuotaURL
	}
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}
	}
	return &QuotaClient{
		refresher:  refresher,
		endpoint:   endpoint,
		httpClient: httpClient,
	}
}

type rawQuotaResponse struct {
	Buckets []rawBucketInfo `json:"buckets"`
}

type rawBucketInfo struct {
	RemainingFraction      *float64        `json:"remainingFraction"`
	RemainingFractionSnake *float64        `json:"remaining_fraction"`
	RemainingAmount        *int64          `json:"remainingAmount"`
	RemainingAmountSnake   *int64          `json:"remaining_amount"`
	ResetTimeRaw           json.RawMessage `json:"resetTime"`
	ResetTimeSnakeRaw      json.RawMessage `json:"reset_time"`
	TokenType              string          `json:"tokenType"`
	TokenTypeSnake         string          `json:"token_type"`
	ModelID                string          `json:"modelId"`
	ModelIDSnake           string          `json:"model_id"`
}

func parseResetTime(raw json.RawMessage) time.Time {
	if len(raw) == 0 {
		return time.Time{}
	}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		return time.Time{}
	}

	// 1. String representation: "2026-10-17T11:29:41Z"
	if strings.HasPrefix(trimmed, "\"") && strings.HasSuffix(trimmed, "\"") {
		var s string
		if err := json.Unmarshal(raw, &s); err == nil {
			if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
				return t.UTC()
			}
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				return t.UTC()
			}
			if t, err := time.Parse("2006-01-02 15:04:05Z", s); err == nil {
				return t.UTC()
			}
		}
	}

	// 2. Protobuf JSON object: {"seconds": 1792056078, "nanos": 0}
	if strings.HasPrefix(trimmed, "{") {
		var protoTs struct {
			Seconds int64 `json:"seconds"`
			Nanos   int64 `json:"nanos"`
		}
		if err := json.Unmarshal(raw, &protoTs); err == nil && protoTs.Seconds > 0 {
			return time.Unix(protoTs.Seconds, protoTs.Nanos).UTC()
		}
	}

	// 3. Epoch numeric seconds: 1792056078
	var epoch int64
	if err := json.Unmarshal(raw, &epoch); err == nil && epoch > 0 {
		if epoch > 1e11 { // milliseconds
			return time.UnixMilli(epoch).UTC()
		}
		return time.Unix(epoch, 0).UTC()
	}

	return time.Time{}
}

func (r *rawBucketInfo) toModelQuotaBucket() ModelQuotaBucket {
	b := ModelQuotaBucket{}
	if r.ModelID != "" {
		b.ModelID = r.ModelID
	} else {
		b.ModelID = r.ModelIDSnake
	}

	if r.TokenType != "" {
		b.TokenType = r.TokenType
	} else {
		b.TokenType = r.TokenTypeSnake
	}

	if r.RemainingFraction != nil {
		b.RemainingFraction = *r.RemainingFraction
	} else if r.RemainingFractionSnake != nil {
		b.RemainingFraction = *r.RemainingFractionSnake
	} else if r.RemainingAmount != nil {
		b.RemainingAmount = *r.RemainingAmount
		if *r.RemainingAmount == 0 {
			b.RemainingFraction = 0.0
		} else {
			b.RemainingFraction = 1.0
		}
	} else if r.RemainingAmountSnake != nil {
		b.RemainingAmount = *r.RemainingAmountSnake
		if *r.RemainingAmountSnake == 0 {
			b.RemainingFraction = 0.0
		} else {
			b.RemainingFraction = 1.0
		}
	} else {
		b.RemainingFraction = 1.0
	}

	if len(r.ResetTimeRaw) > 0 {
		b.ResetTime = parseResetTime(r.ResetTimeRaw)
	}
	if b.ResetTime.IsZero() && len(r.ResetTimeSnakeRaw) > 0 {
		b.ResetTime = parseResetTime(r.ResetTimeSnakeRaw)
	}

	return b
}

// AggregateBuckets parses raw buckets and aggregates them into the three model families:
// Claude 4.6 (Opus/Sonnet), Gemini 2.5 Pro/Flash, and GPT-OSS.
func AggregateBuckets(rawBuckets []ModelQuotaBucket) (claude, gemini, gpt ModelQuotaBucket, canonicalReset time.Time, status string) {
	claude = ModelQuotaBucket{ModelID: "claude-4-6", RemainingFraction: 1.0}
	gemini = ModelQuotaBucket{ModelID: "gemini-2-5", RemainingFraction: 1.0}
	gpt = ModelQuotaBucket{ModelID: "gpt-oss", RemainingFraction: 1.0}

	hasClaude := false
	hasGemini := false
	hasGPT := false

	for _, b := range rawBuckets {
		lower := strings.ToLower(b.ModelID)
		switch {
		case strings.Contains(lower, "claude"):
			if !hasClaude || b.RemainingFraction < claude.RemainingFraction {
				claude = b
				hasClaude = true
			}
		case strings.Contains(lower, "gemini"):
			if !hasGemini || b.RemainingFraction < gemini.RemainingFraction {
				gemini = b
				hasGemini = true
			}
		case strings.Contains(lower, "gpt") || strings.Contains(lower, "oss"):
			if !hasGPT || b.RemainingFraction < gpt.RemainingFraction {
				gpt = b
				hasGPT = true
			}
		}
	}

	// Canonical reset time: prefer the reset time of the lowest fraction bucket
	minFrac := claude.RemainingFraction
	canonicalReset = claude.ResetTime

	if gemini.RemainingFraction < minFrac {
		minFrac = gemini.RemainingFraction
		if !gemini.ResetTime.IsZero() {
			canonicalReset = gemini.ResetTime
		}
	} else if canonicalReset.IsZero() && !gemini.ResetTime.IsZero() {
		canonicalReset = gemini.ResetTime
	}

	if gpt.RemainingFraction < minFrac {
		minFrac = gpt.RemainingFraction
		if !gpt.ResetTime.IsZero() {
			canonicalReset = gpt.ResetTime
		}
	} else if canonicalReset.IsZero() && !gpt.ResetTime.IsZero() {
		canonicalReset = gpt.ResetTime
	}

	// If canonical reset is still zero, try any bucket with non-zero reset time
	if canonicalReset.IsZero() {
		for _, b := range rawBuckets {
			if !b.ResetTime.IsZero() {
				canonicalReset = b.ResetTime
				break
			}
		}
	}

	// Determine status
	if minFrac < 0.15 {
		if claude.RemainingFraction < 0.15 && gemini.RemainingFraction < 0.15 && gpt.RemainingFraction < 0.15 {
			status = StatusDepleted
		} else if gemini.RemainingFraction < 0.15 {
			status = StatusLowGemini
		} else if claude.RemainingFraction < 0.15 {
			status = StatusLowClaude
		} else {
			status = StatusLowGPT
		}
	} else if minFrac <= 0.50 {
		if gemini.RemainingFraction <= 0.50 && claude.RemainingFraction > 0.50 {
			status = StatusLowGemini
		} else if claude.RemainingFraction <= 0.50 && gemini.RemainingFraction > 0.50 {
			status = StatusLowClaude
		} else if gpt.RemainingFraction <= 0.50 {
			status = StatusLowGPT
		} else {
			status = StatusLow
		}
	} else if claude.RemainingFraction >= 1.0 && gemini.RemainingFraction >= 1.0 && gpt.RemainingFraction >= 1.0 {
		status = StatusPristine
	} else {
		status = StatusPlentiful
	}

	return claude, gemini, gpt, canonicalReset, status
}

// RetrieveUserQuota queries Google Cloud Code for the user's quota telemetry.
func (c *QuotaClient) RetrieveUserQuota(ctx context.Context, refreshToken string) (*AccountQuotaData, error) {
	cleanRefreshToken := strings.TrimSpace(refreshToken)
	if cleanRefreshToken == "" {
		return nil, ErrEmptyRefreshToken
	}
	if c.refresher == nil {
		return nil, ErrNilRefresher
	}

	// 1. Refresh OAuth access token
	tokenResult, err := c.refresher.RefreshToken(ctx, cleanRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to refresh access token: %v", ErrQuotaFetchFailed, err)
	}

	// 2. Build HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBufferString("{}"))
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create http request: %v", ErrQuotaFetchFailed, err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResult.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 3. Execute HTTP request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: http request failed: %v", ErrQuotaFetchFailed, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to read response body: %v", ErrQuotaFetchFailed, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: server returned status %d: %s", ErrQuotaFetchFailed, resp.StatusCode, string(bodyBytes))
	}

	// 4. Parse JSON response
	var rawResp rawQuotaResponse
	if err := json.Unmarshal(bodyBytes, &rawResp); err != nil {
		return nil, fmt.Errorf("%w: failed to parse quota json: %v", ErrQuotaFetchFailed, err)
	}

	// 5. Convert raw buckets
	rawBuckets := make([]ModelQuotaBucket, 0, len(rawResp.Buckets))
	for _, rawB := range rawResp.Buckets {
		rawBuckets = append(rawBuckets, rawB.toModelQuotaBucket())
	}

	// 6. Aggregate into model families
	claude, gemini, gpt, resetTime, status := AggregateBuckets(rawBuckets)

	return &AccountQuotaData{
		Claude:      claude,
		Gemini:      gemini,
		GPT:         gpt,
		RawBuckets:  rawBuckets,
		ResetTime:   resetTime,
		Status:      status,
		LastUpdated: time.Now().UTC(),
	}, nil
}
