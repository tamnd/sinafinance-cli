package sinafinance_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/tamnd/sinafinance-cli/sinafinance"
)

// mockSuggestBody simulates a Sina suggest response containing one US stock entry.
const mockSuggestBody = `var suggestdata="AAPL,AAPL,Apple Inc.,100,0,us_AAPL,us,AAPL,105,3,Apple Inc.,100,0,us_AAPL,;"`

// mockSuggestEmpty simulates an empty suggest response (no candidates).
const mockSuggestEmpty = `var suggestdata=""`

// mockQuoteBody simulates a hq.sinajs.cn real-time quote response for sh600036.
const mockQuoteBody = `var hq_str_sh600036="招商银行,47.500,47.350,47.490,47.990,47.130,47.490,47.500,73199025,3469979289.000,47.480,300,47.490,41700,47.500,100,47.510,100,47.520,100,2024-06-14,15:00:00,00,";`

// mockQuoteEmpty simulates a response where the symbol is not found (empty data).
const mockQuoteEmpty = `var hq_str_sh999999="";`

func newTestConfig(ts *httptest.Server) sinafinance.Config {
	cfg := sinafinance.DefaultConfig()
	cfg.SuggestURL = ts.URL
	cfg.QuoteURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 1
	cfg.Timeout = 5 * time.Second
	return cfg
}

func TestResolveFromSuggest(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockSuggestBody))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	sym, err := c.Resolve(context.Background(), "AAPL", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if sym != "us_aapl" {
		t.Errorf("got %q, want us_aapl", sym)
	}
}

func TestResolveGuessDigits6(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockSuggestEmpty))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	sym, err := c.Resolve(context.Background(), "600036", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if sym != "sh600036" {
		t.Errorf("got %q, want sh600036", sym)
	}
}

func TestResolveGuessDigitsOther(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockSuggestEmpty))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	sym, err := c.Resolve(context.Background(), "000001", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if sym != "sz000001" {
		t.Errorf("got %q, want sz000001", sym)
	}
}

func TestResolveGuessLetters(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockSuggestEmpty))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	sym, err := c.Resolve(context.Background(), "MSFT", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if sym != "us_MSFT" {
		t.Errorf("got %q, want us_MSFT", sym)
	}
}

func TestResolvePrefix(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("suggest API should not be called when prefix is already set")
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	sym, err := c.Resolve(context.Background(), "sh600036", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if sym != "sh600036" {
		t.Errorf("got %q, want sh600036", sym)
	}
}

func TestQuoteParsesFields(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockQuoteBody))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	q, err := c.Quote(context.Background(), "sh600036")
	if err != nil {
		t.Fatal(err)
	}

	if q.Name != "招商银行" {
		t.Errorf("Name = %q, want 招商银行", q.Name)
	}
	if q.Price != "47.490" {
		t.Errorf("Price = %q, want 47.490", q.Price)
	}
	if q.PrevClose != "47.350" {
		t.Errorf("PrevClose = %q, want 47.350", q.PrevClose)
	}
	if q.Open != "47.500" {
		t.Errorf("Open = %q, want 47.500", q.Open)
	}
	if q.High != "47.990" {
		t.Errorf("High = %q, want 47.990", q.High)
	}
	if q.Low != "47.130" {
		t.Errorf("Low = %q, want 47.130", q.Low)
	}
	if q.Symbol != "sh600036" {
		t.Errorf("Symbol = %q, want sh600036", q.Symbol)
	}
	// Change should be non-empty (47.490 - 47.350 = 0.140)
	if q.Change == "" {
		t.Error("Change is empty, want a value")
	}
	if q.ChangePct == "" {
		t.Error("ChangePct is empty, want a value")
	}
}

func TestQuoteEmpty(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(mockQuoteEmpty))
	}))
	defer ts.Close()

	cfg := newTestConfig(ts)
	c := sinafinance.NewClient(cfg)

	_, err := c.Quote(context.Background(), "sh999999")
	if err == nil {
		t.Fatal("expected error for empty quote, got nil")
	}
}

func TestGetRetriesOn503(t *testing.T) {
	var hits int
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if hits < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(mockQuoteBody))
	}))
	defer ts.Close()

	cfg := sinafinance.DefaultConfig()
	cfg.QuoteURL = ts.URL
	cfg.SuggestURL = ts.URL
	cfg.Rate = 0
	cfg.Retries = 5
	cfg.Timeout = 5 * time.Second
	c := sinafinance.NewClient(cfg)

	_, err := c.Quote(context.Background(), "sh600036")
	if err != nil {
		t.Fatal(err)
	}
	if hits != 3 {
		t.Errorf("server saw %d hits, want 3", hits)
	}
}
