package sinafinance

import (
	"context"

	"github.com/tamnd/any-cli/kit"
	"github.com/tamnd/any-cli/kit/errs"
)

func init() { kit.Register(Domain{}) }

// Domain is the Sina Finance kit driver.
type Domain struct{}

func (Domain) Info() kit.DomainInfo {
	return kit.DomainInfo{
		Scheme: "sinafinance",
		Hosts:  []string{Host},
		Identity: kit.Identity{
			Binary: "sinafinance",
			Short:  "A command line for Sina Finance stock quotes.",
			Long: `A command line for Sina Finance (finance.sina.com.cn).

Fetches real-time stock quotes for A-shares, US stocks, and Hong Kong stocks
via Sina Finance's unauthenticated quote and suggest APIs.

Accepts plain tickers (AAPL, 600036), Sina-prefixed symbols (sh600036,
sz000001, us_AAPL, hk00700), or company name fragments resolved via the
suggest API. No API key required.

sinafinance is an independent tool and is not affiliated with Sina.`,
			Site: "https://" + Host,
			Repo: "https://github.com/tamnd/sinafinance-cli",
		},
	}
}

func (Domain) Register(app *kit.App) {
	app.SetClient(newKitClient)

	kit.Handle(app, kit.OpMeta{Name: "stock", Group: "quote", Single: true,
		URIType: "stock", Summary: "Get a real-time stock quote",
		Args: []kit.Arg{{Name: "symbol", Help: "ticker, Sina symbol, or company name (e.g. AAPL, 600036, sh600036)"}}}, getStock)
}

func newKitClient(_ context.Context, cfg kit.Config) (any, error) {
	c := DefaultConfig()
	if cfg.UserAgent != "" {
		c.UserAgent = cfg.UserAgent
	}
	if cfg.Rate > 0 {
		c.Rate = cfg.Rate
	}
	if cfg.Retries > 0 {
		c.Retries = cfg.Retries
	}
	if cfg.Timeout > 0 {
		c.Timeout = cfg.Timeout
	}
	return NewClient(c), nil
}

// --- inputs ---

type stockIn struct {
	Symbol string  `kit:"arg" help:"ticker, Sina symbol, or company name (e.g. AAPL, 600036, sh600036)"`
	Market string  `kit:"flag" help:"market hint: auto, sh, sz, us, hk"`
	Client *Client `kit:"inject"`
}

// --- handlers ---

func getStock(ctx context.Context, in stockIn, emit func(*Quote) error) error {
	if in.Symbol == "" {
		return errs.Usage("symbol is required")
	}
	market := in.Market
	if market == "" {
		market = "auto"
	}
	sym, err := in.Client.Resolve(ctx, in.Symbol, market)
	if err != nil {
		return err
	}
	q, err := in.Client.Quote(ctx, sym)
	if err != nil {
		return err
	}
	return emit(q)
}
