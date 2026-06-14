package sinafinance

// Quote holds the fields returned for a single stock symbol.
type Quote struct {
	Symbol    string `json:"symbol"`
	Name      string `json:"name"`
	Price     string `json:"price"`
	Change    string `json:"change"`
	ChangePct string `json:"change_pct"`
	Open      string `json:"open"`
	High      string `json:"high"`
	Low       string `json:"low"`
	PrevClose string `json:"prev_close"`
}
