package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"
	"unsafe"

	"antigravity-switch/internal/connector"

	"golang.org/x/sys/windows"
)

const (
	// DefaultGoogleQuotaURL is the authoritative Google Cloud Code quota endpoint.
	DefaultGoogleQuotaURL = "https://daily-cloudcode-pa.googleapis.com/v1internal:retrieveUserQuota"

	// DefaultUserAgent is the default fallback User-Agent when discovery is inactive.
	DefaultUserAgent = "antigravity/2.24.0"

	// DefaultGoogApiClient is the required Google API client header.
	DefaultGoogApiClient = "gl-go/1.22.0 gdcl/20261017"

	// FallbackUserAgent is the emergency fallback user agent when discovery finds nothing.
	FallbackUserAgent = "antigravity/2.24.0"

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

var (
	reUserAgentName = regexp.MustCompile(`--override_user_agent_name(?:=|\s+)(?:["']?)([a-zA-Z0-9_.-]+)(?:["']?)`)
	reIdeVersion    = regexp.MustCompile(`--override_ide_version(?:=|\s+)(?:["']?)([a-zA-Z0-9_.-]+)(?:["']?)`)
)

var (
	modVersion                  = windows.NewLazySystemDLL("version.dll")
	procGetFileVersionInfoSizeW = modVersion.NewProc("GetFileVersionInfoSizeW")
	procGetFileVersionInfoW     = modVersion.NewProc("GetFileVersionInfoW")
	procVerQueryValueW          = modVersion.NewProc("VerQueryValueW")
)

type vsFixedFileInfo struct {
	Signature        uint32
	StrucVersion     uint32
	FileVersionMS    uint32
	FileVersionLS    uint32
	ProductVersionMS uint32
	ProductVersionLS uint32
	FileFlagsMask    uint32
	FileFlags        uint32
	FileOS           uint32
	FileType         uint32
	FileSubtype      uint32
	FileDateMS       uint32
	FileDateLS       uint32
}

func readPEFileVersion(exePath string) (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("PE version inspection only supported on windows")
	}
	cleanPath := filepath.Clean(strings.TrimSpace(exePath))
	if cleanPath == "" {
		return "", errors.New("empty executable path")
	}
	if _, err := os.Stat(cleanPath); err != nil {
		return "", err
	}

	utfPath, err := windows.UTF16PtrFromString(cleanPath)
	if err != nil {
		return "", err
	}

	var dummy uint32
	size, _, _ := procGetFileVersionInfoSizeW.Call(uintptr(unsafe.Pointer(utfPath)), uintptr(unsafe.Pointer(&dummy)))
	if size == 0 {
		return "", errors.New("GetFileVersionInfoSizeW returned 0")
	}

	buf := make([]byte, size)
	r1, _, _ := procGetFileVersionInfoW.Call(uintptr(unsafe.Pointer(utfPath)), 0, size, uintptr(unsafe.Pointer(&buf[0])))
	if r1 == 0 {
		return "", errors.New("GetFileVersionInfoW failed")
	}

	subBlock, err := windows.UTF16PtrFromString(`\`)
	if err != nil {
		return "", err
	}

	var ptr uintptr
	var length uint32
	r1, _, _ = procVerQueryValueW.Call(
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(subBlock)),
		uintptr(unsafe.Pointer(&ptr)),
		uintptr(unsafe.Pointer(&length)),
	)
	if r1 == 0 || length < uint32(unsafe.Sizeof(vsFixedFileInfo{})) {
		return "", errors.New("VerQueryValueW failed")
	}

	ffi := (*vsFixedFileInfo)(unsafe.Pointer(ptr))
	major := ffi.FileVersionMS >> 16
	minor := ffi.FileVersionMS & 0xFFFF
	build := ffi.FileVersionLS >> 16
	rev := ffi.FileVersionLS & 0xFFFF

	if rev == 0 {
		return fmt.Sprintf("%d.%d.%d", major, minor, build), nil
	}
	return fmt.Sprintf("%d.%d.%d.%d", major, minor, build, rev), nil
}

var (
	cachedUserAgent string
	userAgentMutex  sync.RWMutex
)

// DiscoverAntigravityUserAgent dynamically scans for the running or installed Antigravity version across 3 priority levels.
func DiscoverAntigravityUserAgent() string {
	userAgentMutex.RLock()
	if cachedUserAgent != "" {
		ua := cachedUserAgent
		userAgentMutex.RUnlock()
		return ua
	}
	userAgentMutex.RUnlock()

	userAgentMutex.Lock()
	defer userAgentMutex.Unlock()
	if cachedUserAgent != "" {
		return cachedUserAgent
	}

	cachedUserAgent = resolveAntigravityUserAgent()
	return cachedUserAgent
}

// SetDiscoveredUserAgent overrides the dynamically discovered User-Agent (useful in testing).
func SetDiscoveredUserAgent(ua string) {
	userAgentMutex.Lock()
	defer userAgentMutex.Unlock()
	cachedUserAgent = ua
}

// ResetDiscoveredUserAgent clears the cached User-Agent so the next call performs a fresh scan.
func ResetDiscoveredUserAgent() {
	userAgentMutex.Lock()
	defer userAgentMutex.Unlock()
	cachedUserAgent = ""
}

func resolveAntigravityUserAgent() string {
	// Priority 1: Running language_server.exe process command-line arguments
	if procs, err := connector.QueryLanguageServerProcesses(); err == nil && len(procs) > 0 {
		for _, p := range procs {
			if strings.TrimSpace(p.CommandLine) == "" {
				continue
			}
			verMatches := reIdeVersion.FindStringSubmatch(p.CommandLine)
			if len(verMatches) > 1 && strings.TrimSpace(verMatches[1]) != "" {
				version := strings.TrimSpace(verMatches[1])
				appName := "antigravity"
				nameMatches := reUserAgentName.FindStringSubmatch(p.CommandLine)
				if len(nameMatches) > 1 && strings.TrimSpace(nameMatches[1]) != "" {
					appName = strings.TrimSpace(nameMatches[1])
				}
				return fmt.Sprintf("%s/%s", appName, version)
			}
		}
	}

	// Priority 2: PE FileVersion from Antigravity.exe
	candidatePaths := []string{}
	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		candidatePaths = append(candidatePaths, filepath.Join(localAppData, "Programs", "antigravity", "Antigravity.exe"))
	}
	if progFiles := os.Getenv("ProgramFiles"); progFiles != "" {
		candidatePaths = append(candidatePaths, filepath.Join(progFiles, "Antigravity", "Antigravity.exe"))
	}

	for _, p := range candidatePaths {
		if ver, err := readPEFileVersion(p); err == nil && ver != "" {
			return fmt.Sprintf("antigravity/%s", ver)
		}
	}

	// Priority 3: Fallback safe constant
	return FallbackUserAgent
}

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
	userAgent  string
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

// SetUserAgent sets a custom User-Agent for this client instance (useful for testing).
func (c *QuotaClient) SetUserAgent(ua string) {
	c.userAgent = ua
}

// UserAgent returns the configured or dynamically discovered User-Agent.
func (c *QuotaClient) UserAgent() string {
	if c.userAgent != "" {
		return c.userAgent
	}
	return DiscoverAntigravityUserAgent()
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
	}

	if r.RemainingAmount != nil {
		b.RemainingAmount = *r.RemainingAmount
	} else if r.RemainingAmountSnake != nil {
		b.RemainingAmount = *r.RemainingAmountSnake
	}

	if len(r.ResetTimeRaw) > 0 {
		b.ResetTime = parseResetTime(r.ResetTimeRaw)
	}
	if b.ResetTime.IsZero() && len(r.ResetTimeSnakeRaw) > 0 {
		b.ResetTime = parseResetTime(r.ResetTimeSnakeRaw)
	}

	return b
}

// AggregateBuckets parses raw buckets into standard Claude, Gemini, and GPT families, and computes status.
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
	req.Header.Set("User-Agent", c.UserAgent())
	req.Header.Set("X-Goog-Api-Client", DefaultGoogApiClient)

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

// rawQuotaSummaryResponse represents the Connect-RPC RetrieveUserQuotaSummary structure.
type rawQuotaSummaryResponse struct {
	Response rawQuotaSummaryBody `json:"response"`
}

type rawQuotaSummaryBody struct {
	Groups []rawQuotaGroup `json:"groups"`
}

type rawQuotaGroup struct {
	DisplayName string           `json:"displayName"`
	Description string           `json:"description"`
	Buckets     []rawQuotaBucket `json:"buckets"`
}

type rawQuotaBucket struct {
	BucketID          string   `json:"bucketId"`
	DisplayName       string   `json:"displayName"`
	Window            string   `json:"window"`
	RemainingFraction *float64 `json:"remainingFraction"`
	ResetTime         string   `json:"resetTime"`
}

// ParseQuotaSummary parses the Connect-RPC RetrieveUserQuotaSummary JSON payload.
func ParseQuotaSummary(bodyBytes []byte) (claude, gemini, gpt ModelQuotaBucket, canonicalReset time.Time, status string, err error) {
	var resp rawQuotaSummaryResponse
	if err := json.Unmarshal(bodyBytes, &resp); err != nil {
		return claude, gemini, gpt, canonicalReset, StatusError, fmt.Errorf("failed to parse quota summary JSON: %w", err)
	}

	claude = ModelQuotaBucket{ModelID: "claude-4-6", RemainingFraction: 1.0}
	gemini = ModelQuotaBucket{ModelID: "gemini-2-5", RemainingFraction: 1.0}
	gpt = ModelQuotaBucket{ModelID: "gpt-oss", RemainingFraction: 1.0}

	hasGemini := false
	hasClaudeGPT := false

	for _, g := range resp.Response.Groups {
		lowerGroup := strings.ToLower(g.DisplayName)
		for _, b := range g.Buckets {
			frac := 1.0
			if b.RemainingFraction != nil {
				frac = *b.RemainingFraction
			}
			var resetT time.Time
			if strings.TrimSpace(b.ResetTime) != "" {
				if t, err := time.Parse(time.RFC3339Nano, b.ResetTime); err == nil {
					resetT = t.UTC()
				} else if t, err := time.Parse(time.RFC3339, b.ResetTime); err == nil {
					resetT = t.UTC()
				}
			}

			if strings.Contains(lowerGroup, "gemini") {
				if !hasGemini || frac < gemini.RemainingFraction {
					gemini.RemainingFraction = frac
					gemini.ResetTime = resetT
					gemini.ModelID = b.BucketID
					hasGemini = true
				}
			} else if strings.Contains(lowerGroup, "claude") || strings.Contains(lowerGroup, "gpt") || strings.Contains(lowerGroup, "3p") {
				if !hasClaudeGPT || frac < claude.RemainingFraction {
					claude.RemainingFraction = frac
					claude.ResetTime = resetT
					claude.ModelID = b.BucketID

					gpt.RemainingFraction = frac
					gpt.ResetTime = resetT
					gpt.ModelID = b.BucketID
					hasClaudeGPT = true
				}
			}
		}
	}

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

	return claude, gemini, gpt, canonicalReset, status, nil
}