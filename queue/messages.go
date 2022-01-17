package queue

type EventType string

const (
	EventAccountUpdate EventType = "ACCOUNT_UPDATE"
	EventOrderUpdate   EventType = "ORDER_UPDATE"
)

type OrderStatusType string

const (
	OrderStatusFilled  OrderStatusType = "filled"
	OrderStatusPartial OrderStatusType = "partial"
)

// Telegram
type MessageType string

const (
	MsgTP      MessageType = "takeProfit"
	MsgInfo    MessageType = "info"
	MsgWarning MessageType = "warning"
	MsgError   MessageType = "error"
)

// RabbitMq
type RmqMessageEvent struct {
	BotId    string `json:"botId"`
	UserId   string `json:"userId"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // info | warn
}

type RmqTpEvent struct {
	BotId           string `json:"botId"`
	UserId          string `json:"userId"`
	ExchangeType    string `json:"exchangeType"`
	QuoteAsset      string `json:"quoteAsset"`
	Symbol          string `json:"symbol"`
	EventType       string `json:"eventType"`
	EventSide       string `json:"eventSide"`
	AveragePrice    string `json:"averagePrice"`
	FilledQty       string `json:"filledQty"`
	RealizedProfit  string `json:"realizedProfit"`
	Commission      string `json:"commission"`
	CommissionAsset string `json:"commissionAsset"`
	ExecutedAt      int64  `json:"executedAt"`
	TotalGridSteps  string `json:"totalGridSteps"`
	CurrentGridStep string `json:"currentGridStep"`
}
