package cache

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/tbtc-bot/terrabot-sdk"
	"go.uber.org/zap"
)

type RedisHandler struct {
	Client   *RedisDB
	Logger   *zap.Logger
	Exchange string
}

// STRATEGY
func (rh *RedisHandler) ReadStrategy(session terrabot.Session) (*terrabot.Strategy, error) {

	key := GetRedisKeyStrategy(rh.Exchange, session)
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

	key := GetRedisKeyStrategy(rh.Exchange, session)

	strategyJson, err := json.Marshal(session.Strategy)
	if err != nil {
		return fmt.Errorf("could not marshal strategy with key %s: %s", key, err)
	}
	return rh.Client.Set(key, string(strategyJson))
}

// OPEN ORDERS
func (rh *RedisHandler) ReadOpenOrders(session terrabot.Session) (terrabot.OpenOrders, error) {

	key := GetRedisKeyOpenOrders(rh.Exchange, session)
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

	key := GetRedisKeyOpenOrders(rh.Exchange, session)

	openOrdersJson, err := json.Marshal(openOrders)
	if err != nil {
		return fmt.Errorf("could not marshal open orders with key %s: %s", key, err)
	}

	return rh.Client.Set(key, string(openOrdersJson))
}

// POSITION
func (rh *RedisHandler) ReadPosition(session terrabot.Session) (*terrabot.Position, error) {

	key := GetRedisKeyPosition(rh.Exchange, session)
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

	key := GetRedisKeyPosition(rh.Exchange, session)

	positionJson, err := json.Marshal(position)
	if err != nil {
		return fmt.Errorf("could not marshal position with key %s: %s", key, err)
	}
	return rh.Client.Set(key, string(positionJson))
}

// TP ORDER
func (rh *RedisHandler) ReadTakeProfit(session terrabot.Session) (string, error) {

	key := GetRedisKeyTpId(rh.Exchange, session)
	id, err := rh.Client.Get(key)
	if err != nil {
		return "", fmt.Errorf("could not get from redis take profit ID with key %s: %s", key, err)
	}

	return id, nil
}

func (rh *RedisHandler) WriteTakeProfit(session terrabot.Session, id string) error {

	key := GetRedisKeyTpId(rh.Exchange, session)
	return rh.Client.Set(key, id)
}

func (rh *RedisHandler) DeleteTakeProfit(session terrabot.Session) error {

	key := GetRedisKeyTpId(rh.Exchange, session)
	_, err := rh.Client.Del(key)
	return err
}

// DELETE ALL KEYS
func (rh *RedisHandler) DeleteAllKeys(session terrabot.Session) {

	rh.Client.Del(GetRedisKeyStrategy(rh.Exchange, session))
	rh.Client.Del(GetRedisKeyOpenOrders(rh.Exchange, session))
	rh.Client.Del(GetRedisKeyPosition(rh.Exchange, session))
	rh.Client.Del(GetRedisKeyTpId(rh.Exchange, session))
	rh.Client.Del(GetRedisKeyMetadata(rh.Exchange, session))
	rh.Client.Del(GetRedisKeyGridSize(rh.Exchange, session))

}

// WALLET BALANCE
func (rh *RedisHandler) ReadWalletBalance(botId string) (terrabot.WalletBalance, error) {

	key := GetRedisKeyWalletBalance(rh.Exchange, botId)
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

	key := GetRedisKeyWalletBalance(rh.Exchange, session.BotId)

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

	key := GetRedisKeySymbolInfo(rh.Exchange, symbol)
	exchangeInfoJson, err := rh.Client.Get(key)
	if err != nil {
		return nil, fmt.Errorf("could not read symbol info from Redis")
	}

	var symbolInfo terrabot.SymbolInfo
	if err := json.Unmarshal([]byte(exchangeInfoJson), &symbolInfo); err != nil {
		return nil, fmt.Errorf("could not unmarshal symbol info")
	}

	return &symbolInfo, nil
}

func (rh *RedisHandler) ReadMarkPrice(symbol string) (float64, error) {

	key := GetRedisKeyMarkPrice(rh.Exchange, symbol)
	priceStr, err := rh.Client.Get(key)
	if err != nil {
		return 0, fmt.Errorf("could not get mark price from redis with key %s: %s", key, err)
	}

	return strconv.ParseFloat(priceStr, 64)
}

func (rh *RedisHandler) RedisKey(session terrabot.Session) string {
	return rh.Exchange + "-" + session.BotId + "-" + session.Strategy.Symbol + "-" + string(session.Strategy.PositionSide)
}

// METADATA
func (rh *RedisHandler) ReadMetadata(session terrabot.Session) (*terrabot.Metadata, error) {

	key := GetRedisKeyMetadata(rh.Exchange, session)
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

	key := GetRedisKeyMetadata(rh.Exchange, session)
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

	key := GetRedisKeyGridSize(rh.Exchange, session)
	gridSizeString, err := rh.Client.Get(key)
	if err != nil {
		return 0, fmt.Errorf("could not get from redis grid size with key %s: %s", key, err)
	}
	gridSize, _ := strconv.ParseFloat(gridSizeString, 64)
	return gridSize, nil
}

func (rh *RedisHandler) WriteGridSize(session terrabot.Session, gridSize float64) error {

	key := GetRedisKeyGridSize(rh.Exchange, session)
	return rh.Client.Set(key, fmt.Sprint(gridSize))
}
