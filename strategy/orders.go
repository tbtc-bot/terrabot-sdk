package strategy

import (
	"fmt"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/util"
)

// ORDERS
func (w *StrategyHandler) cancelMultipleOrders(session terrabot.Session, orders []string) error {

	if len(orders) > 0 {

		if err := w.eh.CancelMultipleOrders(session, session.Strategy.Symbol, orders); err != nil {
			return err
		}

		w.removeMultipleOrdersFromMemory(session, orders)
	}

	return nil
}

func (w *StrategyHandler) addGridOrder(session terrabot.Session, order *terrabot.Order) (err error) {
	// round price and size to max symbol precision
	symbolInfo, _ := w.ch.ReadSymbolInfo(order.Symbol)
	pricePrecision := symbolInfo.PricePrecision
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)
	quantityPrecision := symbolInfo.QuantityPrecision
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)

	var id string
	switch order.Type {
	case terrabot.OrderLimit:
		id, err = w.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)
	case terrabot.OrderStopMarket:
		id, err = w.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)
	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}

	// save order to memory
	order.ID = id

	w.addOrderToMemory(session, order)
	return nil
}

func (w *StrategyHandler) addMarketOrder(session terrabot.Session, order *terrabot.Order) error {
	// round size to max symbol precision
	quantityPrecision := w.ch.ReadSymbolQtyPrecision(order.Symbol)
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)
	return w.eh.PlaceOrderMarketRetry(session, order, ATTEMPTS, SLEEP)
}

func (w *StrategyHandler) addTakeProfitOrder(session terrabot.Session, order *terrabot.Order) (err error) {
	// round price with symbol precision
	pricePrecision := w.ch.ReadSymbolPricePrecision(order.Symbol)
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)

	var id string
	switch order.Type {

	case terrabot.OrderLimit:
		id, err = w.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderTrailing:
		id, err = w.eh.PlaceOrderTrailingRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderStopMarket:
		id, err = w.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)

	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}
	order.ID = id

	return nil
}

func (w *StrategyHandler) getOpenOrders(session terrabot.Session) ([]string, error) {

	// first try binance, then redis
	var openOrders []string

	openOrders, err := w.eh.GetOpenOrders(session, session.Strategy.Symbol, session.Strategy.PositionSide)
	if err != nil {

		// get open orders from redis
		openOrdersMap, err := w.ch.ReadOpenOrders(session)
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

func (w *StrategyHandler) addOrderToMemory(session terrabot.Session, order *terrabot.Order) error {
	key := w.ch.RedisKey(session)
	openOrders, err := w.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}
	openOrders[order.ID] = *order

	if err = w.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}

func (w *StrategyHandler) removeMultipleOrdersFromMemory(session terrabot.Session, ids []string) error {
	key := w.ch.RedisKey(session)
	openOrders, err := w.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}

	for _, id := range ids {
		delete(openOrders, id)
	}

	if err = w.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}
