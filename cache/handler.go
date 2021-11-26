package cache

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tbtc-bot/terrabot-sdk"
	"go.uber.org/zap"
)

const (
	KEY_STRATEGY_SUFFIX    = "-strategy"
	KEY_OPEN_ORDERS_SUFFIX = "-openOrders"
	KEY_POSITION_SUFFIX    = "-position"
	MARK_PRICES_SUFFIX     = "-markPrices"
	KEY_TP_SUFFIX          = "-tpId"
	METADATA_SUFFIX        = "-metadata"
	EXCHANGE_INFO_SUFFIX   = "-exchangeInfo"
	GRID_SIZE_SUFFIX       = "-gridSize"
	KEY_WALLET_BALANCE     = "-walletBalance"
)

type RedisHandler struct {
	Client   *RedisDB
	Logger   *zap.Logger
	Exchange string
}

// STRATEGY
func (rh *RedisHandler) ReadStrategy(session terrabot.Session) (*terrabot.Strategy, error) {
	prefixKey := rh.RedisKey(session)
	key := prefixKey + KEY_STRATEGY_SUFFIX
	strategyJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not get from redis strategy with key %s: %s", key, err)
	}

	var strategy terrabot.Strategy
	if err := json.Unmarshal([]byte(strategyJson), &strategy); err != nil {

		rh.Logger.Error("strategy from redis: error unmarshaling",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("key", key),
		)

		return nil, fmt.Errorf("could not unmarshal strategy with key %s: %s", key, err)
	}

	return &strategy, nil
}

func (rh *RedisHandler) WriteStrategy(session terrabot.Session) error {
	key := rh.RedisKey(session) + KEY_STRATEGY_SUFFIX

	strategyJson, err := json.Marshal(session.Strategy)
	if err != nil {
		return fmt.Errorf("could not marshal strategy with key %s: %s", key, err)
	}
	return rh.Client.Set(key, string(strategyJson))
}

// OPEN ORDERS
func (rh *RedisHandler) ReadOpenOrders(session terrabot.Session) (terrabot.OpenOrders, error) {
	key := rh.RedisKey(session) + KEY_OPEN_ORDERS_SUFFIX
	openOrdersJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not get from redis open orders with key %s: %s", key, err)
	}

	var openOrders terrabot.OpenOrders
	if err := json.Unmarshal([]byte(openOrdersJson), &openOrders); err != nil {
		return nil, fmt.Errorf("could not unmarshal open orders with key %s: %s", key, err)
	}

	return openOrders, nil
}

func (rh *RedisHandler) WriteOpenOrders(session terrabot.Session, openOrders terrabot.OpenOrders) error {
	key := rh.RedisKey(session) + KEY_OPEN_ORDERS_SUFFIX

	openOrdersJson, err := json.Marshal(openOrders)
	if err != nil {
		return fmt.Errorf("could not marshal open orders with key %s: %s", key, err)
	}

	return rh.Client.Set(key, string(openOrdersJson))
}

// POSITION
func (rh *RedisHandler) ReadPosition(session terrabot.Session) (*terrabot.Position, error) {
	key := rh.RedisKey(session) + KEY_POSITION_SUFFIX
	positionJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not get from redis position with key %s: %s", key, err)
	}

	var position terrabot.Position
	if err := json.Unmarshal([]byte(positionJson), &position); err != nil {

		rh.Logger.Error("position from redis: error unmarshaling",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("key", key),
		)

		return nil, fmt.Errorf("could not unmarshal position with key %s: %s", key, err)
	}

	return &position, nil
}

func (rh *RedisHandler) WritePosition(session terrabot.Session, position terrabot.Position) error {
	key := rh.RedisKey(session) + KEY_POSITION_SUFFIX

	positionJson, err := json.Marshal(position)
	if err != nil {
		return fmt.Errorf("could not marshal position with key %s: %s", key, err)
	}
	return rh.Client.Set(key, string(positionJson))
}

// TP ORDER
func (rh *RedisHandler) ReadTakeProfit(session terrabot.Session) (string, error) {
	key := rh.RedisKey(session) + KEY_TP_SUFFIX
	id, err := rh.Client.Get(key)
	if err != nil {
		return "", fmt.Errorf("could not get from redis take profit ID with key %s: %s", key, err)
	}

	return id, nil
}

func (rh *RedisHandler) WriteTakeProfit(session terrabot.Session, id string) error {
	key := rh.RedisKey(session) + KEY_TP_SUFFIX

	return rh.Client.Set(key, id)
}

func (rh *RedisHandler) DeleteTakeProfit(session terrabot.Session) error {
	key := rh.RedisKey(session) + KEY_TP_SUFFIX

	_, err := rh.Client.Del(key)
	return err
}

// DELETE ALL KEYS
func (rh *RedisHandler) DeleteAllKeys(session terrabot.Session) {
	baseKey := rh.RedisKey(session)
	rh.Client.Del(baseKey + KEY_STRATEGY_SUFFIX)
	rh.Client.Del(baseKey + KEY_OPEN_ORDERS_SUFFIX)
	rh.Client.Del(baseKey + KEY_POSITION_SUFFIX)
	rh.Client.Del(baseKey + KEY_TP_SUFFIX)
	rh.Client.Del(baseKey + METADATA_SUFFIX)
	rh.Client.Del(baseKey + GRID_SIZE_SUFFIX)
}

// WALLET BALANCE
func (rh *RedisHandler) ReadWalletBalance(botId string) (terrabot.WalletBalance, error) {
	key := rh.Exchange + "-" + botId + KEY_WALLET_BALANCE
	balanceJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not get from redis balance with key %s: %s", key, err)
	}

	var balance terrabot.WalletBalance
	if err := json.Unmarshal([]byte(balanceJson), &balance); err != nil {
		return nil, fmt.Errorf("could not unmarshal balance with key %s: %s", key, err)
	}

	return balance, nil
}

func (rh *RedisHandler) WriteWalletBalance(session terrabot.Session, walletBalance terrabot.WalletBalance) error {
	key := rh.Exchange + "-" + session.BotId + KEY_WALLET_BALANCE

	balanceJson, err := json.Marshal(walletBalance)
	if err != nil {
		return fmt.Errorf("could not marshal wallet balance with key %s: %s", key, err)
	}
	return rh.Client.Set(key, string(balanceJson))
}

// EXCHANGE INFO
func (rh *RedisHandler) ReadSymbolPricePrecision(symbol string) int {
	symbolInfo, err := rh.ReadSymbolInfo(symbol)
	if err != nil {
		rh.Logger.Error("Could not get symbol price precision from Redis",
			zap.String("error", err.Error()),
			zap.String("symbol", symbol),
		)
		return 1
	}
	return symbolInfo.PricePrecision
}

func (rh *RedisHandler) ReadSymbolQtyPrecision(symbol string) int {
	symbolInfo, err := rh.ReadSymbolInfo(symbol)
	if err != nil {
		rh.Logger.Error("Could not get symbol qty precision from Redis",
			zap.String("error", err.Error()),
			zap.String("symbol", symbol),
		)
		return 1
	}
	return symbolInfo.QuantityPrecision
}

func (rh *RedisHandler) ReadSymbolMinQty(symbol string) float64 {
	symbolInfo, err := rh.ReadSymbolInfo(symbol)
	if err != nil {
		rh.Logger.Error("Could not get symbol min qty from Redis",
			zap.String("error", err.Error()),
			zap.String("symbol", symbol),
		)
		return 1
	}
	return symbolInfo.MinQty
}

func (rh *RedisHandler) ReadSymbolInfo(symbol string) (*terrabot.SymbolInfo, error) {
	exchangeInfoJson, err := rh.Client.Get(rh.Exchange + EXCHANGE_INFO_SUFFIX)
	if err != nil {
		return nil, fmt.Errorf("could not read exchange info from Redis")
	}
	var exchangeInfo terrabot.ExchangeInfo
	if err := json.Unmarshal([]byte(exchangeInfoJson), &exchangeInfo); err != nil {
		return nil, fmt.Errorf("could not unmarshal exchange info")
	}
	symbolInfo, ok := exchangeInfo[symbol]
	if !ok {
		return nil, fmt.Errorf("could not find symbol %s in exchange info", symbol)
	}
	return &symbolInfo, nil
}

func (rh *RedisHandler) ReadMarkPrice(symbol string) (float64, error) {
	key := rh.Exchange + MARK_PRICES_SUFFIX
	prices, err := rh.Client.Get(key)
	if err != nil {
		return 0, fmt.Errorf("coulf not get mark price from redis with key %s: %s", key, err)
	}

	var pricesMap map[string]string
	if err := json.Unmarshal([]byte(prices), &pricesMap); err != nil {
		return 0, fmt.Errorf("could not unmarshal prices with key %s: %s", key, err)
	}

	priceStr, ok := pricesMap[symbol]
	if !ok {
		return 0, fmt.Errorf("mark price not found found with key %s", key)
	}

	return strconv.ParseFloat(priceStr, 64)
}

func (rh *RedisHandler) RedisKey(session terrabot.Session) string {
	return rh.Exchange + "-" + session.BotId + "-" + session.Strategy.Symbol + "-" + string(session.Strategy.PositionSide)
}

// METADATA
func (rh *RedisHandler) ReadMetadata(session terrabot.Session) (*terrabot.Metadata, error) {
	key := rh.RedisKey(session) + METADATA_SUFFIX
	metadataJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not get from redis metadata with key %s: %s", key, err)
	}

	var metadata terrabot.Metadata
	if err := json.Unmarshal([]byte(metadataJson), &metadata); err != nil {
		return nil, fmt.Errorf("could not unmarshal metadata with key %s: %s", key, err)
	}

	return &metadata, nil
}

func (rh *RedisHandler) WriteMetadata(session terrabot.Session, metadata terrabot.Metadata) {
	key := rh.RedisKey(session) + METADATA_SUFFIX
	metadataJson, err := json.Marshal(metadata)
	if err != nil {

		rh.Logger.Error("Could not marshal metadata",
			zap.String("last grid reached", fmt.Sprint(metadata.LastGridReached)),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return
	}

	if err = rh.Client.Set(key, string(metadataJson)); err != nil {

		rh.Logger.Error("Could store metadata in redis",
			zap.String("last grid reached", fmt.Sprint(metadata.LastGridReached)),
			zap.String("key", key),
			zap.String("error", err.Error()),
		)
		return
	}
}

// GRID SIZE
/* Used for TakeStepLimit */
func (rh *RedisHandler) ReadGridSize(session terrabot.Session) (float64, error) {
	key := rh.RedisKey(session) + GRID_SIZE_SUFFIX
	gridSizeString, err := rh.Client.Get(key)
	if err != nil {
		return 0, fmt.Errorf("could not get from redis grid size with key %s: %s", key, err)
	}
	gridSize, _ := strconv.ParseFloat(gridSizeString, 64)
	return gridSize, nil
}

func (rh *RedisHandler) WriteGridSize(session terrabot.Session, gridSize float64) error {
	key := rh.RedisKey(session) + GRID_SIZE_SUFFIX
	return rh.Client.Set(key, fmt.Sprint(gridSize))
}
