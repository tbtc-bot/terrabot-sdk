package strategy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"go.uber.org/zap"
)

func (w *StrategyHandler) ParseCommandEvent(eventRaw []byte) {
	var err error

	var event queue.RmqApiServerCommandEvent
	if err = json.Unmarshal(eventRaw, &event); err != nil {

		w.logger.Error("Error during unmarshaling of command event", zap.String("error", err.Error()))

		return
	}

	w.logger.Info("Received command event", zap.String("command", event.Command))

	// create new session
	strategyTmp := terrabot.NewStrategy(terrabot.StrategyDummy, event.Symbol, terrabot.PositionSideType(event.PositionSide), terrabot.StrategyParameters{}) // dummy strategy without parameters
	session := terrabot.NewSession(event.BotId, event.UserId, event.AccessKey, event.SecretKey, *strategyTmp)

	// acquire mutex lock
	key := w.ch.RedisKey(*session)
	var errLock error
	time.Sleep(50 * time.Millisecond) // TODO change this

	mu := w.ch.Client.GetNewMutex(key)
	if mu == nil {
		msg := fmt.Sprintf("parseCommandEvent: mutex is nil with key %s", key)
		w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  session.UserId,
			BotId:   session.BotId,
			Message: msg,
		})

		w.logger.Error("parseCommandEvent: mutex is nil",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("key", key),
		)
	} else {

		if err := mu.Lock(); err != nil {
			msg := fmt.Sprintf("parseCommandEvent %s: error acquiring lock with key %s: %s", event.Command, key, err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("parseCommandEvent: error acquiring lock",
				zap.String("error", err.Error()),
				zap.String("botId", session.BotId),
				zap.String("command", event.Command),
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
			msg := fmt.Sprintf("parseCommandEvent %s: error releasing lock with key %s: %s", event.Command, key, err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("parseCommandEvent: error releasing lock",
				zap.String("error", err.Error()),
				zap.String("botId", session.BotId),
				zap.String("command", event.Command),
				zap.String("key", key),
			)
		} else if !ok {
			w.logger.Error("parseCommandEvent: error releasing lock",
				zap.String("error", "!ok"),
				zap.String("botId", session.BotId),
				zap.String("command", event.Command),
				zap.String("key", key),
			)
		}
	}(errLock)

	// get strategy parameters
	strategy, err := w.ReadStrategy(*session)
	if err != nil {
		return
	}
	session.Strategy = *strategy

	switch event.Command {
	case "start":
		//TODO return if position is not 0

		// get pars from redis and declare strategy
		if err = w.commandStart(*session); err != nil {
			msg := fmt.Sprintf("Could not START strategy with key %s: %s", w.ch.RedisKey(*session), err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("Could not START strategy",
				zap.String("error", err.Error()),
				zap.String("command", "start"),
				zap.String("key", key))

		}
	case "stop":
		if err = w.commandStop(*session); err != nil {
			msg := fmt.Sprintf("Could not STOP strategy with key %s: %s", w.ch.RedisKey(*session), err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("Could not STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "stop"),
				zap.String("key", key))

		}
	case "hardStop":
		if err = w.commandHardStop(*session); err != nil {
			msg := fmt.Sprintf("Could not HARD_STOP strategy with key %s: %s", w.ch.RedisKey(*session), err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("Could not HARD_STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "hardStop"),
				zap.String("key", key))

		}
	case "softStop":
		if err = w.commandSoftStop(*session); err != nil {
			msg := fmt.Sprintf("Could not SOFT_STOP strategy with key %s: %s", w.ch.RedisKey(*session), err)
			w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			w.logger.Error("Could not SOFT_STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "softStop"),
				zap.String("key", key))
		}
	default:
		msg := fmt.Sprintf("Event command not recognized: %s", event.Command)
		w.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  event.UserId,
			BotId:   event.BotId,
			Message: msg,
		})
	}
}

func (w *StrategyHandler) commandStart(session terrabot.Session) (err error) {
	lastStatus := session.Strategy.Status

	session.Strategy.Status = terrabot.StatusStart
	if err := w.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", w.ch.RedisKey(session), err)
	}

	// if the status is soft stop, cancel it and put it to start
	if lastStatus == terrabot.StatusSoftStop {

		if err := w.ch.WriteStrategy(session); err != nil {
			return fmt.Errorf("could not store strategy with key %s: %s", w.ch.RedisKey(session), err)
		}

		return w.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStart)

	} else {

		// update position
		position := terrabot.Position{Symbol: session.Strategy.Symbol, PositionSide: session.Strategy.PositionSide, EntryPrice: 0, Size: 0, MarkPrice: 0}
		if err = w.ch.WritePosition(session, position); err != nil {
			return fmt.Errorf("could not store position in redis with key %s: %s", w.ch.RedisKey(session), err)
		}

		// update open orders
		if err = w.ch.WriteOpenOrders(session, terrabot.OpenOrders{}); err != nil {
			return fmt.Errorf("could not store open orders in redis with key %s: %s", w.ch.RedisKey(session), err)
		}

		err = w.StartStrategy(session)
		if err != nil {
			return err
		}

		return w.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStart)
	}
}

func (w *StrategyHandler) commandStop(session terrabot.Session) error {
	session.Strategy.Status = terrabot.StatusStop

	if err := w.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", w.ch.RedisKey(session), err)
	}

	if err := w.cancelAllOpenOrders(session); err != nil {
		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	return w.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStop)
}

func (w *StrategyHandler) commandHardStop(session terrabot.Session) error {
	session.Strategy.Status = terrabot.StatusHardStop

	if err := w.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", w.ch.RedisKey(session), err)
	}

	if err := w.cancelAllOpenOrders(session); err != nil {
		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	if err := w.ClosePosition(session); err != nil {
		return fmt.Errorf("could not close position: %s", err)
	}

	err := w.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusHardStop)
	if err != nil {
		return err
	}

	w.ch.DeleteAllKeys(session)
	return nil
}

func (w *StrategyHandler) commandSoftStop(session terrabot.Session) error {
	if session.Strategy.Status == terrabot.StatusStart {
		session.Strategy.Status = terrabot.StatusSoftStop
	} else {
		return fmt.Errorf("status is already %s", session.Strategy.Status)
	}

	key := w.ch.RedisKey(session)
	if err := w.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", key, err)
	}

	return w.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusSoftStop)
}

func (w *StrategyHandler) cancelAllOpenOrders(session terrabot.Session) error {

	openOrders, err := w.getOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not cancel open orders - could not get open orders: %s", err)
	}

	if err := w.cancelMultipleOrders(session, openOrders); err != nil {

		w.logger.Error("Could not cancel open orders",
			zap.String("error", err.Error()),
			zap.String("key", w.ch.RedisKey(session)),
			zap.String("open orders list", fmt.Sprint(openOrders)),
		)

		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	return nil
}
