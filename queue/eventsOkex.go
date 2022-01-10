package queue

type RmqProbeEvent struct {
	UserId         string      `json:"userId"`
	BotId          string      `json:"botId"`
	AccessKey      string      `json:"accessKey"`
	SecretKey      string      `json:"secretKey"`
	Passphrase     string      `json:"passphrase,omitempty"`
	SimulationMode bool        `json:"simulationMode,omitempty"`
	EventType      EventType   `json:"eventType"`
	EventTime      string      `json:"eventTime"`
	Data           interface{} `json:"data"`
}

type RmqAccountUpdateData struct {
	Balances  *[]RmqAccountUpdateBalance
	Positions *[]RmqAccountUpdatePosition
}

type RmqAccountUpdateBalance struct {
	Asset        string  `json:"asset"`
	AssetBalance float64 `json:"assetBalance"`
}

type RmqAccountUpdatePosition struct {
	Symbol         string  `json:"symbol"`
	InstrumentType string  `json:"instrumentType"`
	PositionAmount float64 `json:"positionAmount"`
	PositionSide   string  `json:"positionSide"`
	EntryPrice     float64 `json:"entryPrice"`
}

type RmqOrderUpdateData struct {
	Symbol         string  `json:"symbol"`
	InstrumentType string  `json:"instrumentType"`
	OrderId        string  `json:"orderId"`
	TradeTime      string  `json:"tradeTime"`
	PositionSide   string  `json:"positionSide"`
	FilledQuantity float64 `json:"filledQuantity"`
	OrderStatus    string  `json:"orderStatus"`
	OriginalPrice  float64 `json:"originalPrice"`
	ExecutionPrice float64 `json:"executionPrice"`
	RealizedPnl    float64 `json:"realizedPnl"`
	Tag            string  `json:"tag"`
}
