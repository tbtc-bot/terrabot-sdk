package strategy

import (
	"fmt"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/util"
)

// ORDERS
func (s *Strategy) cancelMultipleOrders(session terrabot.Session, orders []string) error {

	if len(orders) > 0 {

		if err := s.eh.CancelMultipleOrders(session, session.Strategy.Symbol, orders); err != nil {
			return err
		}

		s.removeMultipleOrdersFromMemory(session, orders)
	}

	return nil
}

func (s *Strategy) addGridOrder(session terrabot.Session, order *terrabot.Order) (err error) {
	// round price and size to max symbol precision
	symbolInfo, _ := s.ch.ReadSymbolInfo(order.Symbol)
	pricePrecision := symbolInfo.PricePrecision
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)
	quantityPrecision := symbolInfo.QuantityPrecision
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)

	var id string
	switch order.Type {
	case terrabot.OrderLimit:
		id, err = s.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)
	case terrabot.OrderStopMarket:
		id, err = s.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)
	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}

	// save order to memory
	order.ID = id

	s.addOrderToMemory(session, order)
	return nil
}

func (s *Strategy) addMarketOrder(session terrabot.Session, order *terrabot.Order) error {
	// TODO round precision should be done in the exchange api requests

	// round size to max symbol precision
	quantityPrecision := s.ch.ReadSymbolQtyPrecision(order.Symbol)
	order.Amount = util.RoundFloatWithPrecision(order.Amount, quantityPrecision)
	return s.eh.PlaceOrderMarketRetry(session, order, ATTEMPTS, SLEEP)
}

func (s *Strategy) addTakeProfitOrder(session terrabot.Session, order *terrabot.Order) (err error) {
	// TODO round precision should be done in the exchange api requests

	// round price with symbol precision
	pricePrecision := s.ch.ReadSymbolPricePrecision(order.Symbol)
	order.Price = util.RoundFloatWithPrecision(order.Price, pricePrecision)
	order.TriggerPrice = util.RoundFloatWithPrecision(order.TriggerPrice, pricePrecision)

	var id string
	switch order.Type {

	case terrabot.OrderLimit:
		id, err = s.eh.PlaceOrderLimitRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderTrailing:
		id, err = s.eh.PlaceOrderTrailingRetry(session, order, ATTEMPTS, SLEEP)

	case terrabot.OrderStopMarket:
		id, err = s.eh.PlaceOrderStopMarketRetry(session, order, ATTEMPTS, SLEEP)

	default:
		return fmt.Errorf("order type %s not valid for take profit order", order.Type)
	}
	if err != nil {
		return err
	}
	order.ID = id

	return nil
}

func (s *Strategy) getOpenOrders(session terrabot.Session) ([]string, error) {

	// first try binance, then redis
	var openOrders []string

	openOrders, err := s.eh.GetOpenOrders(session, session.Strategy.Symbol, session.Strategy.PositionSide)
	if err != nil {

		// get open orders from redis
		openOrdersMap, err := s.ch.ReadOpenOrders(session)
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

func (s *Strategy) addOrderToMemory(session terrabot.Session, order *terrabot.Order) error {
	key := s.ch.RedisKey(session)
	openOrders, err := s.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}
	openOrders[order.ID] = *order

	if err = s.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}

func (s *Strategy) removeMultipleOrdersFromMemory(session terrabot.Session, ids []string) error {
	key := s.ch.RedisKey(session)
	openOrders, err := s.ch.ReadOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders with key %s: %s", key, err)
	}

	for _, id := range ids {
		delete(openOrders, id)
	}

	if err = s.ch.WriteOpenOrders(session, openOrders); err != nil {
		return fmt.Errorf("could not set open orders with key %s: %s", key, err)
	}
	return nil
}
