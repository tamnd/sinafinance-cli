package sinafinance

// Quote holds the fields returned for a single stock symbol.
type Quote struct {
	Symbol    string `json:"symbol"    kit:"id" table:"symbol"`
	Name      string `json:"name"               table:"name"`
	Price     string `json:"price"              table:"price"`
	Change    string `json:"change"             table:"change"`
	ChangePct string `json:"change_pct"         table:"change%"`
	Open      string `json:"open"               table:"open"`
	High      string `json:"high"               table:"high"`
	Low       string `json:"low"                table:"low"`
	PrevClose string `json:"prev_close"         table:"prev_close"`
}
