package exchange

import (
	"time"

	"github.com/tbtc-bot/terrabot-sdk"
	"go.uber.org/zap"
)

type ExchangeConnector interface {
	GetMarkPrice(symbol string) (float64, error)
	GetMarkPriceRetry(symbol string, attempts int, sleep time.Duration) (price float64, err error)

	CancelOrder(session terrabot.Session, symbol string, orderID string) error
	CancelOrderRetry(session terrabot.Session, symbol string, orderID string, attempts int, sleep time.Duration) (err error)

	CancelMultipleOrders(session terrabot.Session, symbol string, orderIDList []string) error

	PlaceOrderLimit(session terrabot.Session, order *terrabot.Order) (string, error)
	PlaceOrderLimitRetry(session terrabot.Session, order *terrabot.Order, attempts int, sleep time.Duration) (orderID string, err error)

	PlaceOrderMarket(session terrabot.Session, order *terrabot.Order) error
	PlaceOrderMarketRetry(session terrabot.Session, order *terrabot.Order, attempts int, sleep time.Duration) (err error)

	PlaceOrderStopMarket(session terrabot.Session, order *terrabot.Order) (orderID string, err error)
	PlaceOrderStopMarketRetry(session terrabot.Session, order *terrabot.Order, attempts int, sleep time.Duration) (orderID string, err error)

	PlaceOrderTrailing(session terrabot.Session, order *terrabot.Order) (orderID string, err error)
	PlaceOrderTrailingRetry(session terrabot.Session, order *terrabot.Order, attempts int, sleep time.Duration) (orderID string, err error)

	GetOpenOrders(session terrabot.Session, symbol string, positionSide terrabot.PositionSideType) ([]string, error)
	GetPositionAmount(session terrabot.Session, symbol string, positionSide terrabot.PositionSideType) (float64, error)
	ClosePosition(session terrabot.Session, symbol string, currency string) error

	GetBalance(session terrabot.Session) (terrabot.WalletBalance, error)
	GetBalanceRetry(session terrabot.Session, attempts int, sleep time.Duration) (balance terrabot.WalletBalance, err error)

	SetLogger(logger *zap.Logger)
}
