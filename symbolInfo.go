package terrabot

type SymbolInfo struct {
	Pair                  string  `json:"pair,omitempty"`
	MaintMarginPercent    float64 `json:"maintMarginPercent,omitempty"`
	RequiredMarginPercent float64 `json:"requiredMarginPercent,omitempty"`
	MinQty                float64 `json:"minQty,omitempty"`
	MinNotionalQty        float64 `json:"minNotionalQty,omitempty"`
	PricePrecision        int     `json:"pricePrecision,omitempty"`
	QuantityPrecision     int     `json:"quantityPrecision,omitempty"`
	BaseAssetPrecision    int     `json:"baseAssetPrecision,omitempty"`
	QuotePrecision        int     `json:"quotePrecision,omitempty"`
	QuoteAsset            string  `json:"quoteAsset,omitempty"`
	MarginAsset           string  `json:"marginAsset,omitempty"`
	BaseAsset             string  `json:"baseAsset,omitempty"`
}
