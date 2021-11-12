package queue

type RmqUserDataEvent struct {
	UserId    string                 `json:"userId"`
	BotId     string                 `json:"botId"`
	Body      map[string]interface{} `json:"event"`
	AccessKey string                 `json:"accessKey"`
	SecretKey string                 `json:"secretKey"`
}

type RmqApiServerCommandEvent struct {
	BotId        string `json:"botId"`
	UserId       string `json:"userId"`
	AccessKey    string `json:"accessKey"`
	SecretKey    string `json:"secretKey"`
	Command      string `json:"command"`
	Symbol       string `json:"symbol"`
	PositionSide string `json:"positionSide"`
}

type WsAccountUpdate struct {
	Reason    string       `json:"m"` // required
	Balances  []WsBalance  `json:"B"`
	Positions []WsPosition `json:"P"` // required
}

type WsBalance struct {
	Asset              string `json:"a"`
	Balance            string `json:"wb"`
	CrossWalletBalance string `json:"cw"`
	ChangeBalance      string `json:"bc"`
}

type WsPosition struct {
	Symbol                    string `json:"s"`  // required
	Side                      string `json:"ps"` // required
	Amount                    string `json:"pa"` // required
	MarginType                string `json:"mt"`
	IsolatedWallet            string `json:"iw"`
	EntryPrice                string `json:"ep"` // required
	MarkPrice                 string `json:"mp"`
	UnrealizedPnL             string `json:"up"`
	AccumulatedRealized       string `json:"cr"`
	MaintenanceMarginRequired string `json:"mm"`
}

type WsOrderTradeUpdate struct {
	Symbol               string `json:"s"` // required
	ClientOrderID        string `json:"c"`
	Side                 string `json:"S"`
	Type                 string `json:"o"`
	TimeInForce          string `json:"f"`
	OriginalQty          string `json:"q"`
	OriginalPrice        string `json:"p"`  // required
	AveragePrice         string `json:"ap"` // required
	StopPrice            string `json:"sp"`
	ExecutionType        string `json:"x"`
	Status               string `json:"X"` // required
	ID                   int64  `json:"i"` // required
	LastFilledQty        string `json:"l"` // required
	AccumulatedFilledQty string `json:"z"`
	LastFilledPrice      string `json:"L"`
	CommissionAsset      string `json:"N"`
	Commission           string `json:"n"`
	TradeTime            int64  `json:"T"` // required
	TradeID              int64  `json:"t"`
	BidsNotional         string `json:"b"`
	AsksNotional         string `json:"a"`
	IsMaker              bool   `json:"m"`
	IsReduceOnly         bool   `json:"R"`
	WorkingType          string `json:"wt"`
	OriginalType         string `json:"ot"`
	PositionSide         string `json:"ps"` // required
	IsClosingPosition    bool   `json:"cp"`
	ActivationPrice      string `json:"AP"`
	CallbackRate         string `json:"cr"`
	RealizedPnL          string `json:"rp"` // required
}
