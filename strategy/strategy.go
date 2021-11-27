package strategy

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/cache"
	"github.com/tbtc-bot/terrabot-sdk/database"
	"github.com/tbtc-bot/terrabot-sdk/exchange"
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"github.com/tbtc-bot/terrabot-sdk/telegram"
	"github.com/tbtc-bot/terrabot-sdk/util"
	"go.uber.org/zap"
)

const (
	// retry parameters
	ATTEMPTS = 3
	SLEEP    = 200 * time.Millisecond
)

type StrategyHandler struct {
	ch     *cache.RedisHandler
	dh     *database.FirestoreHandler
	eh     exchange.ExchangeHandler
	th     *telegram.TelegramHandler
	logger *zap.Logger
}

func NewStrategyHandler(ch *cache.RedisHandler, dh *database.FirestoreHandler, eh exchange.ExchangeHandler, th *telegram.TelegramHandler, logger *zap.Logger) *StrategyHandler {
	return &StrategyHandler{
		ch:     ch,
		dh:     dh,
		eh:     eh,
		th:     th,
		logger: logger,
	}
}

// Handle a Binance AccountUpdate event
func (w *StrategyHandler) HandleAccountUpdate(ctx context.Context, session terrabot.Session, event *queue.WsAccountUpdate) {

	for _, wsPos := range event.Positions {
		position := w.newPositionFromWsPosition(&wsPos) // convert from binance struct to Position struct
		if position != nil {
			session.Strategy = terrabot.Strategy{
				Symbol:       position.Symbol,
				PositionSide: position.PositionSide,
				Status:       "",
				Parameters:   terrabot.StrategyParameters{},
			}
			if err := w.handlePositionUpdate(session, *position); err != nil {
				w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
					UserId:  session.UserId,
					BotId:   session.BotId,
					Message: err.Error(),
				})

				w.logger.Error("Could not handle position update",
					zap.String("botId", session.BotId),
					zap.String("error", err.Error()),
					zap.String("key", w.ch.RedisKey(session)),
				)
			}
		}
	}
}

func (w *StrategyHandler) handlePositionUpdate(session terrabot.Session, position terrabot.Position) (err error) {
	time.Sleep(50 * time.Millisecond) // TODO change this

	// acquire mutex lock
	key := w.ch.RedisKey(session)
	mu := w.ch.Client.GetNewMutex(key)
	if mu == nil {
		msg := fmt.Sprintf("handlePositionUpdate: mutex is nil with key %s", key)
		w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  session.UserId,
			BotId:   session.BotId,
			Message: msg,
		})

		w.logger.Error("handlePositionUpdate: mutex is nil",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("key", key),
		)
	} else {

		if err = mu.Lock(); err != nil {
			msg := fmt.Sprintf("handlePositionUpdate: error acquiring lock with key %s: %s", key, err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  session.UserId,
				BotId:   session.BotId,
				Message: msg,
			})

			w.logger.Error("handlePositionUpdate: error acquiring lock",
				zap.String("botId", session.BotId),
				zap.String("error", err.Error()),
				zap.String("key", key),
			)
		}
	}

	// release mutex lock
	defer func(errLock error) {
		// do nothing if lock failed
		if errLock != nil {
			return
		}

		ok, err := mu.Unlock()
		if err != nil {

			msg := fmt.Sprintf("handlePositionUpdate: error releasing lock with key %s: %s", key, err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  session.UserId,
				BotId:   session.BotId,
				Message: msg,
			})

			w.logger.Error("handlePositionUpdate: error releasing lock",
				zap.String("botId", session.BotId),
				zap.String("error", err.Error()),
				zap.String("key", key),
			)
		} else if !ok {
			w.logger.Error("handlePositionUpdate: error releasing lock",
				zap.String("botId", session.BotId),
				zap.String("error", "!ok"),
				zap.String("key", key),
			)
		}
	}(err)

	// get last position from redis
	lastPosition, err := w.ch.ReadPosition(session)
	if err != nil {
		// if position is not found, it has been deleted by hard stop command
		return nil
	}

	// compare new position with old position
	if lastPosition.EntryPrice != position.EntryPrice || lastPosition.Size != position.Size {
		// store new position
		if err = w.ch.WritePosition(session, position); err != nil {
			return fmt.Errorf("could not store position with key %s: %s", key, err)
		}

		if err = w.ExecuteSession(session, position); err != nil {
			return fmt.Errorf("could not execute session: %s", err)
		}
	}

	return nil
}

func (w *StrategyHandler) HandleOrderUpdate(ctx context.Context, session terrabot.Session, event *queue.WsOrderTradeUpdate) {

	session.Strategy = terrabot.Strategy{
		Symbol:       event.Symbol,
		PositionSide: terrabot.PositionSideType(event.PositionSide),
	}

	strategy, err := w.ReadStrategy(session)
	if err != nil {

		strategy = &terrabot.Strategy{
			Symbol:       event.Symbol,
			PositionSide: terrabot.PositionSideType(event.PositionSide),
		}
	}
	session.Strategy = *strategy

	//
	switch event.Status {

	case "NEW":
		break

	case "CANCELED", "EXPIRED":
		break

	case "FILLED":
		profit, err := strconv.ParseFloat(event.RealizedPnL, 64)
		if err != nil {
			w.logger.Error("could not parse string to float",
				zap.String("key", w.ch.RedisKey(session)),
				zap.String("profit", event.RealizedPnL),
			)
		}

		if profit != 0 {
			lastGridReached := w.lastGridReached(session)

			tp := queue.RmqTpEvent{
				BotId:           session.BotId,
				UserId:          session.UserId,
				Symbol:          event.Symbol,
				EventType:       "full",
				EventSide:       string(event.PositionSide),
				AveragePrice:    event.AveragePrice,
				FilledQty:       event.LastFilledQty,
				RealizedProfit:  event.RealizedPnL,
				ExecutedAt:      event.TradeTime,
				TotalGridSteps:  strconv.FormatUint(uint64(session.Strategy.Parameters.GridOrders), 10),
				CurrentGridStep: lastGridReached,
			}

			w.th.SendTelegramTP(tp)

			metadata, err := w.ch.ReadMetadata(session)
			if err == nil {
				metadata.LastGridReached = 0
				w.ch.WriteMetadata(session, *metadata)
			}

			// check if soft stop
			strategy, err := w.ReadStrategy(session)
			if err == nil {
				if strategy.Status == terrabot.StatusSoftStop {
					msg := fmt.Sprintf("SOFT STOP %s realized profit (%s/%s) %s USDT",
						event.Symbol, lastGridReached, fmt.Sprint(strategy.Parameters.GridOrders), event.RealizedPnL)

					w.th.SendTelegramMessage(queue.MsgInfo, queue.RmqMessageEvent{
						UserId:  session.UserId,
						BotId:   session.BotId,
						Message: msg,
					})
				}
			}

		} else {
			// update last grid reached
			metadata, err := w.ch.ReadMetadata(session)
			if err != nil {

				if string(session.Strategy.PositionSide) != "BOTH" { // TODO change this
					w.logger.Warn("Could not update last grid reached",
						zap.String("error", err.Error()))
				}

				return
			}

			lastGridReached, ok := metadata.MapIDtoGridNumber[fmt.Sprint(event.ID)]
			if !ok {
				return
			}
			metadata.LastGridReached = lastGridReached
			w.ch.WriteMetadata(session, *metadata)

			// set stop loss if last grid reached
			if metadata.LastGridReached == int64(session.Strategy.Parameters.GridOrders) {
				// notify last grid reached

				price, _ := strconv.ParseFloat(event.OriginalPrice, 64)
				sym := session.Strategy.Symbol + "-" + fmt.Sprintf("%c", string(session.Strategy.PositionSide)[0])
				msg := fmt.Sprintf("%s last grid reached at price %f USDT", sym, price) // TODO add current PNL and liquidation price

				w.th.SendTelegramMessage(queue.MsgInfo, queue.RmqMessageEvent{
					UserId:  session.UserId,
					BotId:   session.BotId,
					Message: msg,
				})
			}
		}

	case "PARTIALLY_FILLED":
		profit, _ := strconv.ParseFloat(event.RealizedPnL, 64)
		lastGridReached := w.lastGridReached(session)
		if profit != 0 {

			tp := queue.RmqTpEvent{
				BotId:           session.BotId,
				UserId:          session.UserId,
				Symbol:          event.Symbol,
				EventType:       "partial",
				EventSide:       string(event.PositionSide),
				AveragePrice:    event.AveragePrice,
				FilledQty:       event.LastFilledQty,
				RealizedProfit:  event.RealizedPnL,
				ExecutedAt:      event.TradeTime,
				TotalGridSteps:  strconv.FormatUint(uint64(session.Strategy.Parameters.GridOrders), 10),
				CurrentGridStep: lastGridReached,
			}

			w.th.SendTelegramTP(tp)
		}

	default:
		w.logger.Warn("Event type not recognized",
			zap.String("event", string(event.Status)))

	}
}

func (w *StrategyHandler) newPositionFromWsPosition(p *queue.WsPosition) *terrabot.Position {

	amount, _ := strconv.ParseFloat(p.Amount, 64)
	entryPrice, _ := strconv.ParseFloat(p.EntryPrice, 64)
	markPrice, err := w.GetMarkPrice(p.Symbol)
	if err != nil {
		// TODO change this
		markPrice, _ = strconv.ParseFloat(p.MarkPrice, 64)
	}

	return &terrabot.Position{
		Symbol:       p.Symbol,
		PositionSide: terrabot.PositionSideType(p.Side),
		Size:         amount,
		EntryPrice:   entryPrice,
		MarkPrice:    markPrice,
	}
}

func (w *StrategyHandler) lastGridReached(session terrabot.Session) (lastGrid string) {
	/* Get from Redis the last grid reached, in a string form, for the telegram notification. Use "-" if not found. */

	metadata, err := w.ch.ReadMetadata(session)
	if err != nil {
		lastGrid = "0"
	} else {
		lastGrid = fmt.Sprint(metadata.LastGridReached)
	}
	return
}

func (w *StrategyHandler) ExecuteSession(session terrabot.Session, position terrabot.Position) (err error) {

	strategy, err := w.ReadStrategy(session)
	if err != nil {
		return nil // ignore if strategy not found
	}
	session.Strategy = *strategy

	if position.Size == 0 {
		return w.StartStrategy(session)

	} else {
		// position size is not zero: set take profit
		w.CancelLastTakeProfit(session)

		// do nothing if the status is stop or hard stop
		if session.Strategy.Status == terrabot.StatusStop || session.Strategy.Status == terrabot.StatusHardStop {
			return nil
		}

		if err = w.SetTakeProfit(session, position); err != nil {
			positionSize, _ := w.eh.GetPositionAmount(session, position.Symbol, position.PositionSide)

			if positionSize == 0 {
				return w.StartStrategy(session)
			}

			w.logger.Error("Error in set take profit",
				zap.String("botId", session.BotId),
				zap.String("symbol", session.Strategy.Symbol),
				zap.String("positionSide", string(session.Strategy.PositionSide)),
				zap.String("updatePosition", fmt.Sprint(position.Size)),
				zap.String("binancePosition", fmt.Sprint(positionSize)),
				zap.String("error", err.Error()),
			)

			position.Size = math.Abs(positionSize)
			if err = w.SetTakeProfit(session, position); err != nil {
				msg := fmt.Sprintf("%s - could not place take profit order: %s", w.ch.RedisKey(session), err)
				w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
					UserId:  session.UserId,
					BotId:   session.BotId,
					Message: msg,
				})
				return fmt.Errorf("could not set take profit: %s", err)
			}
		}
	}
	return nil
}

func (w *StrategyHandler) StartStrategy(session terrabot.Session) (err error) {

	// do nothing if status is not start
	if session.Strategy.Status != terrabot.StatusStart {
		w.ch.DeleteTakeProfit(session)
		w.commandHardStop(session)
		return nil
	}

	symbol := session.Strategy.Symbol
	asset, err := util.GetAssetFromSymbol(symbol)
	if err != nil {
		return fmt.Errorf("could not get base asset: %s", err)
	}
	balance, err := w.getAssetBalance(session, terrabot.Asset(asset))
	if err != nil {
		return fmt.Errorf("could not get wallet balance: %s", err)
	}

	markPrice, err := w.GetMarkPrice(symbol)
	if err != nil {
		return fmt.Errorf("%s could not get mark price for symbol %s: %s", session.BotId, symbol, err)
	}

	////////////////////////////////////////////////////////////////////////
	pars := session.Strategy.Parameters
	var s0 float64 // initial order size of the token
	if pars.OrderBaseType == terrabot.OrderBaseTypePerc {
		s0 = (balance / markPrice) * (pars.OrderSize / 100)

	} else if pars.OrderBaseType == terrabot.OrderBaseTypeFix {
		s0 = pars.OrderSize / markPrice

	} else {
		return fmt.Errorf("invalid order base type %s", pars.OrderBaseType)
	}

	// check minimum order
	quantityPrecision := w.ch.ReadSymbolQtyPrecision(symbol)
	s0 = util.RoundFloatWithPrecision(s0, quantityPrecision) // initial order size
	s0usd := s0 * markPrice                                  // initial order size in dollars
	if s0usd < 5 {
		w.commandHardStop(session)
		return fmt.Errorf("initial order after rounding is %f %s, but it must be at least 5 %s", s0usd, asset, asset)
	}

	minQty := w.ch.ReadSymbolMinQty(symbol)
	if s0 < minQty {
		w.commandHardStop(session)
		return fmt.Errorf("initial order after rounding is %f %s, but it must be at least %f %s", s0usd, symbol, minQty, symbol)
	}
	////////////////////////////////////////////////////////////////////////

	// execute market order
	var order *terrabot.Order
	switch session.Strategy.PositionSide {

	case terrabot.PositionSideLong:
		order = terrabot.NewOrderMarket(symbol, terrabot.SideBuy, terrabot.PositionSideLong, s0)
		if err = w.addMarketOrder(session, order); err != nil {
			return fmt.Errorf("%s could not add market order %s: %s", session.BotId, order.String(), err)
		}

	case terrabot.PositionSideShort:
		order = terrabot.NewOrderMarket(symbol, terrabot.SideSell, terrabot.PositionSideShort, s0)
		if err = w.addMarketOrder(session, order); err != nil {
			return fmt.Errorf("%s could not add market order %s: %s", session.BotId, order.String(), err)
		}
	}

	// this is for TakeStepLimit
	if err := w.StoreGridSize(session, s0); err != nil {
		w.logger.Warn("Could not store grid size in redis",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
		)
		msg := fmt.Sprintf("Could not store grid size in redis: %s", err)
		w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  session.UserId,
			BotId:   session.BotId,
			Message: msg,
		})
	}

	// create grid
	if err = w.CreateGrid(session, balance, markPrice); err != nil {
		return fmt.Errorf("could not create grid: %s", err)
	}

	return nil
}

func (w *StrategyHandler) StoreGridSize(session terrabot.Session, s0 float64) error {
	GridOrders := float64(session.Strategy.Parameters.GridOrders)
	OrderFactor := session.Strategy.Parameters.OrderFactor
	gridSize := s0 * math.Pow(OrderFactor, GridOrders)
	return w.ch.WriteGridSize(session, gridSize)
}

func (w *StrategyHandler) CreateGrid(session terrabot.Session, balance float64, startPrice float64) error {

	if err := w.CancelGrid(session); err != nil {
		return fmt.Errorf("could not cancel grid: %s", err)
	}

	// create orders
	orders, err := session.Strategy.GridOrders(balance, startPrice)
	if err != nil {
		return fmt.Errorf("could not create grid orders: %s", err)
	}

	mapIDtoGridNumber := make(map[string]int64)

	// execute orders
	for _, order := range orders {
		if err := w.addGridOrder(session, order); err != nil {
			msg := fmt.Sprintf("could not place grid order %s: %s", order.String(), err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  session.UserId,
				BotId:   session.BotId,
				Message: msg,
			})

			w.logger.Error("Could not place grid order",
				zap.String("botId", session.BotId),
				zap.String("error", err.Error()),
				zap.String("order", order.String()),
				zap.Float64("walletBalance", balance),
				zap.Float64("startPrice", startPrice),
				zap.Float64("orderSize", session.Strategy.Parameters.OrderSize),
			)

		} else {
			mapIDtoGridNumber[order.ID] = order.GridNumber
		}
	}

	// update only mapIDtoGridNumber, keep last grid reached for tp message
	metadata, err := w.ch.ReadMetadata(session)
	if err != nil {
		metadata = &terrabot.Metadata{
			LastGridReached: 0,
		}
	}
	metadata.MapIDtoGridNumber = mapIDtoGridNumber
	w.ch.WriteMetadata(session, *metadata)
	return nil
}

func (w *StrategyHandler) CancelGrid(session terrabot.Session) error {

	openOrders, err := w.getOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not get open orders: %s", err)
	}

	if err := w.cancelMultipleOrders(session, openOrders); err != nil {
		return err
	}

	w.CancelLastTakeProfit(session)
	return nil
}

func (w *StrategyHandler) SetTakeProfit(session terrabot.Session, position terrabot.Position) (err error) {
	w.CancelLastTakeProfit(session)

	gridSize, err := w.ch.ReadGridSize(session)
	if err != nil {
		w.logger.Error("Could not get grid size from redis",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
		)
		msg := fmt.Sprintf("Could not get grid size from redis: %s", err)
		w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  session.UserId,
			BotId:   session.BotId,
			Message: msg,
		})
		gridSize = 0
	}
	order, err := session.Strategy.TakeProfitOrder(position, gridSize)
	if err != nil {
		return fmt.Errorf("could not create take profit order: %s", err)
	}

	if err = w.addTakeProfitOrder(session, order); err != nil {
		return fmt.Errorf("could not place take profit order %s: %s", order.String(), err)
	}

	if err := w.ch.WriteTakeProfit(session, order.ID); err != nil {
		w.logger.Error("Could not store take profit order in Redis",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("orderId", order.String()))
	}
	return nil
}

func (w *StrategyHandler) CancelLastTakeProfit(session terrabot.Session) {
	id, err := w.ch.ReadTakeProfit(session)
	if err != nil {
		return
	}
	if err := w.eh.CancelOrderRetry(session, session.Strategy.Symbol, id, ATTEMPTS, SLEEP); err == nil {
		w.ch.DeleteTakeProfit(session)
	}
}

func (w *StrategyHandler) ClosePosition(session terrabot.Session) error {
	positionAmount, err := w.eh.GetPositionAmount(session, session.Strategy.Symbol, session.Strategy.PositionSide)
	if err != nil {
		return fmt.Errorf("could not get position amount: %s", err)
	}

	// do nothing if the position is already zero
	if positionAmount == 0 {
		return nil
	}

	// opposite order side to close position
	var orderSide terrabot.SideType
	if util.ComparePositionSides(string(session.Strategy.PositionSide), string(terrabot.PositionSideLong)) {
		orderSide = terrabot.SideSell
	} else if util.ComparePositionSides(string(session.Strategy.PositionSide), string(terrabot.PositionSideShort)) {
		orderSide = terrabot.SideBuy
	} else {
		return fmt.Errorf("position side %s not recognized", session.Strategy.PositionSide)
	}

	order := terrabot.NewOrderMarket(session.Strategy.Symbol, orderSide, session.Strategy.PositionSide, math.Abs(positionAmount))
	if err = w.addMarketOrder(session, order); err != nil {
		return fmt.Errorf("%s could not add market order %s: %s", session.BotId, order.String(), err)
	}
	// TODO delete position key from redis
	return nil
}

func (w *StrategyHandler) GetMarkPrice(symbol string) (float64, error) {
	markPrice, err := w.ch.ReadMarkPrice(symbol)
	if err != nil {
		w.logger.Warn("Could not get mark price from Redis",
			zap.String("error", err.Error()),
			zap.String("symbol", symbol))
		return w.eh.GetMarkPriceRetry(symbol, ATTEMPTS, SLEEP)
	}
	return markPrice, nil
}

func (w *StrategyHandler) ReadStrategy(session terrabot.Session) (strategy *terrabot.Strategy, err error) {

	// read from redis first
	strategy, err = w.ch.ReadStrategy(session)
	if err == nil {
		return strategy, nil
	}

	// in case of error check on firestore
	strategy, err = w.dh.ReadStrategy(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide)
	if err != nil {
		return nil, fmt.Errorf("could not get strategy from redis nor firestore with key %s: %s", w.ch.RedisKey(session), err)
	}

	// store on redis
	session.Strategy = *strategy
	w.ch.WriteStrategy(session)

	return strategy, nil
}
