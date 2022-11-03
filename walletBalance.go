package terrabot

type Balance struct {
	AccountAlias       string `json:"accountAlias"`
	Asset              string `json:"asset"`
	Balance            string `json:"balance"`
	CrossWalletBalance string `json:"crossWalletBalance"`
	CrossUnPnl         string `json:"crossUnPnl"`
	AvailableBalance   string `json:"availableBalance"`
	MaxWithdrawAmount  string `json:"maxWithdrawAmount"`
}

type WalletBalance map[string]*Balance

type Asset string

const (
	USDT Asset = "USDT"
	BUSD Asset = "BUSD"
	BNB  Asset = "BNB"
)
