package strategy

import (
	"fmt"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/util"
)

// ORDERS
func (sh *StrategyHandler) cancelMultipleOrders(session terrabot.Session, orders []string) error {

	if len(orders) > 0 {

		if err := sh.eh.CancelMultipleOrders(session, session.Strategy.Symbol, orders); err != nil {
			return err
		}

		sh.removeMultipleOrdersFromMemory(session, orders)
	}

	return nil
}

func (sh *StrategyHandler) addGridOrder(session terrabot.Session, order *terrabot.Order) (err error) {

	// round price and size to max symbol precision
	symbolInfo, _ := sh.ch.ReadSymbolInfo(order.Symbol)
	pricePrecision := symbolInfo.PricePrecision
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)
	quantityPrecision := symbolInfo.QuantityPrecision
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)

	var id string
	switch order.Type {
	case terrabot.OrderLimit:
		id, err = sh.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)
	case terrabot.OrderStopMarket:
		id, err = sh.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)
	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}

	// save order to memory
	order.ID = id

	sh.addOrderToMemory(session, order)
	return nil
}

func (sh *StrategyHandler) addMarketOrder(session terrabot.Session, order *terrabot.Order) error {
	// TODO round precision should be done in the exchange api requests
	// round size to max symbol precision
	quantityPrecision := sh.ch.ReadSymbolQtyPrecision(order.Symbol)
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)
	return sh.eh.PlaceOrderMarketRetry(session, order, ATTEMPTS, SLEEP)
}

func (sh *StrategyHandler) addTakeProfitOrder(session terrabot.Session, order *terrabot.Order) (err error) {
	// TODO round precision should be done in the exchange api requests
	// round price with symbol precision
	pricePrecision := sh.ch.ReadSymbolPricePrecision(order.Symbol)
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)

	var id string
	switch order.Type {

	case terrabot.OrderLimit:
		id, err = sh.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderTrailing:
		id, err = sh.eh.PlaceOrderTrailingRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderStopMarket:
		id, err = sh.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)

	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}
	order.ID = id

	return nil
}

func (sh *StrategyHandler) getOpenOrders(session terrabot.Session) ([]string, error) {

	// first try binance, then redis
	var openOrders []string

	openOrders, err := sh.eh.GetOpenOrders(session, session.Strategy.Symbol, session.Strategy.PositionSide)
	if err != nil {

		// get open orders from redis
		openOrdersMap, err := sh.ch.ReadOpenOrders(session)
		if err != nil {
			return nil, fmt.Errorf("could not get open orders neither from Binance and Redis: %s", err)
		}

		openOrders = make([]string, 0, len(openOrdersMap))
		for _, o := range openOrdersMap {
			openOrders = append(openOrders, o.ID)
		}

	}

	return openOrders, nil
}

func (sh *StrategyHandler) addOrderToMemory(session terrabot.Session, order *terrabot.Order) error {

	key := sh.ch.RedisKey(session)
	openOrders, err := sh.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}
	openOrders[order.ID] = *order

	if err = sh.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}

func (sh *StrategyHandler) removeMultipleOrdersFromMemory(session terrabot.Session, ids []string) error {

	key := sh.ch.RedisKey(session)
	openOrders, err := sh.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}

	for _, id := range ids {
		delete(openOrders, id)
	}

	if err = sh.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}
