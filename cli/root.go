// Package cli builds the sinafinance command tree on top of the sinafinance library.
package cli

import (
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
	"github.com/tamnd/sinafinance-cli/pkg/render"
	"github.com/tamnd/sinafinance-cli/sinafinance"
)

// Build metadata, set via -ldflags at release time.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// exit codes.
const (
	exitError  = 1
	exitUsage  = 2
	exitNoData = 3
)

// ExitError carries a process exit code up to main.
type ExitError struct {
	Code int
	Err  error
}

func (e *ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return fmt.Sprintf("exit %d", e.Code)
}

func (e *ExitError) Unwrap() error { return e.Err }

func codeError(code int, err error) error { return &ExitError{Code: code, Err: err} }

// App holds shared state threaded through every command.
type App struct {
	client *sinafinance.Client
	cfg    sinafinance.Config

	output   string
	fields   []string
	noHeader bool
	template string
	quiet    bool
}

// Root builds the root command and its subtree.
func Root() *cobra.Command {
	app := &App{cfg: sinafinance.DefaultConfig()}

	root := &cobra.Command{
		Use:   "sinafinance",
		Short: "Get stock quotes from Sina Finance",
		Long: `sinafinance fetches real-time stock quotes from Sina Finance.

No API key is required. Accepts plain tickers (AAPL, 600036) or
Sina-prefixed symbols (sh600036, sz000001, us_AAPL).

sinafinance is an independent tool and is not affiliated with Sina.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return app.setup()
		},
	}

	pf := root.PersistentFlags()
	pf.StringVarP(&app.output, "output", "o", "auto", "output: table|json|jsonl|csv|tsv (auto=table on TTY, jsonl piped)")
	pf.StringSliceVar(&app.fields, "fields", nil, "comma-separated columns to include")
	pf.BoolVar(&app.noHeader, "no-header", false, "omit the header row in table/csv/tsv")
	pf.StringVar(&app.template, "template", "", "Go text/template applied per record")
	pf.BoolVarP(&app.quiet, "quiet", "q", false, "suppress progress on stderr")

	pf.DurationVar(&app.cfg.Rate, "delay", app.cfg.Rate, "minimum spacing between requests")
	pf.DurationVar(&app.cfg.Timeout, "timeout", app.cfg.Timeout, "per-request timeout")
	pf.IntVar(&app.cfg.Retries, "retries", app.cfg.Retries, "retry attempts on 429/5xx")
	pf.StringVar(&app.cfg.UserAgent, "user-agent", app.cfg.UserAgent, "User-Agent sent with each request")

	root.AddCommand(
		app.stockCmd(),
		newVersionCmd(),
	)
	return root
}

func (a *App) setup() error {
	if a.output == "" || a.output == "auto" {
		if isatty.IsTerminal(os.Stdout.Fd()) {
			a.output = string(render.FormatTable)
		} else {
			a.output = string(render.FormatJSONL)
		}
	}
	if !render.Format(a.output).Valid() {
		return codeError(exitUsage, fmt.Errorf("unknown output format %q", a.output))
	}
	a.client = sinafinance.NewClient(a.cfg)
	return nil
}

func (a *App) render(records any) error {
	r := render.New(os.Stdout, render.Format(a.output), a.fields, a.noHeader, a.template)
	return r.Render(records)
}

func (a *App) progressf(format string, args ...any) {
	if a.quiet {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}
