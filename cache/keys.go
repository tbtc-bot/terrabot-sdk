package cache

import (
	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/data_types"
)

const (
	MARK_PRICE  = "markPrice"
	SYMBOL_INFO = "symbolInfo"

	STRATEGY    = "strategy"
	OPEN_ORDERS = "openOrders"
	POSITION    = "position"
	TP_ID       = "tpId"
	METADATA    = "metadata"

	WALLET_BALANCE = "walletBalance"
)

//
func GetRedisKeyMarkPrice(exchange string, symbol string) string {
	return exchange + "-" + MARK_PRICE + "-" + symbol // e.g. binancef-markPrice-BTCUSDT
}

func GetRedisKeySymbolInfo(exchange string, symbol string) string {
	return exchange + "-" + SYMBOL_INFO + "-" + symbol // e.g. binancef-symbolInfo-BTCUSDT
}

func GetRedisKeyWalletBalance(exchange string, botId string) string {
	return exchange + "-" + botId + "-" + WALLET_BALANCE // e.g. binancef-botId-walletBalance
}

//
func GetRedisKeyStrategy(exchange string, session terrabot.Session) string {
	return GetRedisKeyBase(exchange, session) + "-" + STRATEGY // e.g. binancef-botId-BTCUSDT-LONG-strategy
}

func GetRedisKeyOpenOrders(exchange string, session terrabot.Session) string {
	return GetRedisKeyBase(exchange, session) + "-" + OPEN_ORDERS // e.g. binancef-botId-BTCUSDT-LONG-openOrders
}

func GetRedisKeyPosition(exchange string, session terrabot.Session) string {
	return GetRedisKeyBase(exchange, session) + "-" + POSITION // e.g. binancef-botId-BTCUSDT-LONG-position
}

func GetRedisKeyTpId(exchange string, session terrabot.Session) string {
	return GetRedisKeyBase(exchange, session) + "-" + TP_ID // e.g. binancef-botId-BTCUSDT-LONG-tpId
}

func GetRedisKeyMetadata(exchange string, session terrabot.Session) string {
	return GetRedisKeyBase(exchange, session) + "-" + METADATA // e.g. binancef-botId-BTCUSDT-LONG-metadata
}

func GetRedisKeyBase(exchange string, session terrabot.Session) string {

	switch exchange {

	case data_types.BinanceFutures:
		return exchange + "-" + session.BotId + "-" + session.Strategy.Symbol + "-" + string(session.Strategy.PositionSide) // e.g. binancef-botId-BTCUSDT-LONG

	case data_types.OkexMargin, data_types.OkexFutures:
		return exchange + "-" + session.BotId + "-" + session.Strategy.Symbol // e.g. okexf-botId-BTCUSDT

	default:
		return ""
	}
}
