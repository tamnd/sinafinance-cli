// Package sinafinance is the library behind the sinafinance command: the HTTP
// client, symbol resolution, and quote fetching for Sina Finance.
//
// The client talks to two unauthenticated Sina Finance APIs:
//   - suggest3.sinajs.cn for symbol resolution
//   - hq.sinajs.cn for real-time quotes
//
// It paces requests, retries transient 429/5xx errors with exponential
// backoff, and sets the Referer header required by hq.sinajs.cn.
package sinafinance

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Host is the canonical hostname for Sina Finance.
const Host = "finance.sina.com.cn"

// DefaultSuggestURL is the Sina suggest endpoint.
const DefaultSuggestURL = "https://suggest3.sinajs.cn"

// DefaultQuoteURL is the Sina real-time quote endpoint.
const DefaultQuoteURL = "https://hq.sinajs.cn"

// DefaultUserAgent identifies the client to Sina Finance.
const DefaultUserAgent = "sinafinance/dev (+https://github.com/tamnd/sinafinance-cli)"

// Config holds constructor parameters.
type Config struct {
	SuggestURL string
	QuoteURL   string
	UserAgent  string
	Rate       time.Duration
	Timeout    time.Duration
	Retries    int
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		SuggestURL: DefaultSuggestURL,
		QuoteURL:   DefaultQuoteURL,
		UserAgent:  DefaultUserAgent,
		Rate:       200 * time.Millisecond,
		Timeout:    30 * time.Second,
		Retries:    3,
	}
}

// Client talks to the Sina Finance APIs.
type Client struct {
	cfg        Config
	httpClient *http.Client
	mu         sync.Mutex
	last       time.Time
}

// NewClient returns a Client with the given config.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: cfg.Timeout},
	}
}

// Resolve returns the Sina internal symbol for key (e.g. "sh600036", "us_AAPL").
// market is "auto", "sh", "sz", or "us". If market is "auto" the function first
// tries the suggest API and falls back to prefix guessing.
func (c *Client) Resolve(ctx context.Context, key, market string) (string, error) {
	// Already has a recognised prefix — use as-is.
	upper := strings.ToUpper(key)
	if strings.HasPrefix(upper, "SH") || strings.HasPrefix(upper, "SZ") ||
		strings.HasPrefix(upper, "HK") || strings.HasPrefix(upper, "US_") {
		return strings.ToLower(key), nil
	}

	// Try suggest API.
	suggestURL := fmt.Sprintf("%s/suggest/type=11,12,13,14,15&key=%s&name=suggestdata",
		c.cfg.SuggestURL, url.QueryEscape(key))
	body, err := c.get(ctx, suggestURL, "")
	if err == nil {
		if sym := parseSuggest(string(body)); sym != "" {
			return sym, nil
		}
	}

	// Fall back to guessing.
	return guessSymbol(key, market), nil
}

// parseSuggest extracts the first resolved symbol from a Sina suggest response.
// Response format: var suggestdata="entry1;entry2;..."
// Each entry: code,code,name,type,?,sinaSymbol,...
func parseSuggest(resp string) string {
	start := strings.Index(resp, `"`)
	if start < 0 {
		return ""
	}
	resp = resp[start+1:]
	end := strings.LastIndex(resp, `"`)
	if end <= 0 {
		return ""
	}
	resp = resp[:end]

	for _, part := range strings.Split(resp, ";") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		fields := strings.Split(part, ",")
		if len(fields) >= 6 && fields[5] != "" {
			return strings.ToLower(fields[5])
		}
	}
	return ""
}

// guessSymbol applies heuristic prefixing when the suggest API is unavailable.
func guessSymbol(key, market string) string {
	lower := strings.ToLower(key)
	switch market {
	case "sh":
		return "sh" + lower
	case "sz":
		return "sz" + lower
	case "us":
		return "us_" + strings.ToUpper(key)
	}
	// auto: pure digits → A-share, letters → US stock
	allDigits := true
	for _, ch := range lower {
		if ch < '0' || ch > '9' {
			allDigits = false
			break
		}
	}
	if allDigits {
		if strings.HasPrefix(lower, "6") {
			return "sh" + lower
		}
		return "sz" + lower
	}
	return "us_" + strings.ToUpper(key)
}

// Quote fetches the real-time quote for an already-resolved Sina symbol.
func (c *Client) Quote(ctx context.Context, symbol string) (*Quote, error) {
	quoteURL := fmt.Sprintf("%s/list=%s", c.cfg.QuoteURL, symbol)
	body, err := c.get(ctx, quoteURL, "https://finance.sina.com.cn/")
	if err != nil {
		return nil, err
	}

	line := string(body)
	start := strings.Index(line, `"`)
	end := strings.LastIndex(line, `"`)
	if start < 0 || end <= start {
		return nil, fmt.Errorf("unexpected response: %s", trunc(line, 100))
	}
	data := line[start+1 : end]
	fields := strings.Split(data, ",")

	if len(fields) < 6 || fields[0] == "" {
		return nil, fmt.Errorf("symbol %s not found or market closed", symbol)
	}

	name := fields[0]
	open := fields[1]
	prevClose := fields[2]
	now := fields[3]
	high := fields[4]
	low := fields[5]

	change := ""
	changePct := ""
	nowF := parseFloat(now)
	prevF := parseFloat(prevClose)
	if prevF != 0 {
		diff := nowF - prevF
		change = fmt.Sprintf("%.3f", diff)
		changePct = fmt.Sprintf("%.2f%%", diff/prevF*100)
	}

	return &Quote{
		Symbol:    symbol,
		Name:      name,
		Price:     now,
		Change:    change,
		ChangePct: changePct,
		Open:      open,
		High:      high,
		Low:       low,
		PrevClose: prevClose,
	}, nil
}

// get performs a GET request with optional Referer, pacing, and retries.
func (c *Client) get(ctx context.Context, rawURL, referer string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= c.cfg.Retries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff(attempt)):
			}
		}
		b, retry, err := c.do(ctx, rawURL, referer)
		if err == nil {
			return b, nil
		}
		lastErr = err
		if !retry {
			return nil, err
		}
	}
	return nil, fmt.Errorf("get %s: %w", rawURL, lastErr)
}

func (c *Client) do(ctx context.Context, rawURL, referer string) ([]byte, bool, error) {
	c.pace()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, false, err
	}
	req.Header.Set("User-Agent", c.cfg.UserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, true, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return nil, true, fmt.Errorf("http %d", resp.StatusCode)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("http %d", resp.StatusCode)
	}

	b, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, true, err
	}
	return b, false, nil
}

func (c *Client) pace() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cfg.Rate <= 0 {
		return
	}
	if wait := c.cfg.Rate - time.Since(c.last); wait > 0 {
		time.Sleep(wait)
	}
	c.last = time.Now()
}

func backoff(attempt int) time.Duration {
	d := time.Duration(attempt) * 500 * time.Millisecond
	if d > 5*time.Second {
		d = 5 * time.Second
	}
	return d
}

func parseFloat(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(s, "%f", &f)
	return f
}

func trunc(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
