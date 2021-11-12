package terrabot

type SymbolInfo struct {
	Pair                  string  `json:"pair"`
	MaintMarginPercent    float64 `json:"maintMarginPercent"`
	RequiredMarginPercent float64 `json:"requiredMarginPercent"`
	MinQty                float64 `json:"minQty"`
	MinNotionalQty        float64 `json:"minNotionalQty"`
	PricePrecision        int     `json:"pricePrecision"`
	QuantityPrecision     int     `json:"quantityPrecision"`
	BaseAssetPrecision    int     `json:"baseAssetPrecision"`
	QuotePrecision        int     `json:"quotePrecision"`
	QuoteAsset            string  `json:"quoteAsset"`
	MarginAsset           string  `json:"marginAsset"`
	BaseAsset             string  `json:"baseAsset"`
}

type ExchangeInfo map[string]SymbolInfo
