package cli

import (
	"github.com/spf13/cobra"
	"github.com/tamnd/sinafinance-cli/sinafinance"
)

func (a *App) stockCmd() *cobra.Command {
	var market string
	cmd := &cobra.Command{
		Use:   "stock <symbol>",
		Short: "Get a real-time stock quote (e.g. AAPL, 600036, sz000001)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			a.progressf("resolving %s...", key)
			sym, err := a.client.Resolve(cmd.Context(), key, market)
			if err != nil {
				return codeError(exitError, err)
			}

			a.progressf("fetching quote for %s...", sym)
			q, err := a.client.Quote(cmd.Context(), sym)
			if err != nil {
				return codeError(exitError, err)
			}
			if q == nil {
				return codeError(exitNoData, nil)
			}

			return a.render([]*sinafinance.Quote{q})
		},
	}
	cmd.Flags().StringVar(&market, "market", "auto", "Market hint: auto, sh, sz, us")
	return cmd
}
