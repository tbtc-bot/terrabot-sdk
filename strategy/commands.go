package strategy

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"go.uber.org/zap"
)

func (s *Strategy) ParseCommandEvent(eventRaw []byte) {
	var err error

	var event queue.RmqApiServerCommandEvent
	if err = json.Unmarshal(eventRaw, &event); err != nil {

		s.logger.Error("Error during unmarshaling of command event", zap.String("error", err.Error()))

		return
	}

	s.logger.Info("Received command event", zap.String("command", event.Command))

	// create new session
	strategyTmp := terrabot.NewStrategy(terrabot.StrategyDummy, event.Symbol, terrabot.PositionSideType(event.PositionSide), terrabot.StrategyParameters{}) // dummy strategy without parameters
	session := terrabot.NewSession(event.BotId, event.UserId, event.AccessKey, event.SecretKey, event.PassPhrase, event.SimulationMode, *strategyTmp)

	// acquire mutex lock
	key := s.ch.RedisKey(*session)
	var errLock error
	time.Sleep(50 * time.Millisecond) // TODO change this

	mu := s.ch.Client.GetNewMutex(key)
	if mu == nil {
		msg := fmt.Sprintf("parseCommandEvent: mutex is nil with key %s", key)
		s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  session.UserId,
			BotId:   session.BotId,
			Message: msg,
		})

		s.logger.Error("parseCommandEvent: mutex is nil",
			zap.String("botId", session.BotId),
			zap.String("error", err.Error()),
			zap.String("key", key),
		)
	} else {

		if err := mu.Lock(); err != nil {
			msg := fmt.Sprintf("parseCommandEvent %s: error acquiring lock with key %s: %s", event.Command, key, err)
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("parseCommandEvent: error acquiring lock",
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
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("parseCommandEvent: error releasing lock",
				zap.String("error", err.Error()),
				zap.String("botId", session.BotId),
				zap.String("command", event.Command),
				zap.String("key", key),
			)
		} else if !ok {
			s.logger.Error("parseCommandEvent: error releasing lock",
				zap.String("error", "!ok"),
				zap.String("botId", session.BotId),
				zap.String("command", event.Command),
				zap.String("key", key),
			)
		}
	}(errLock)

	// get strategy parameters
	strategy, err := s.ReadStrategy(*session)
	if err != nil {
		return
	}
	session.Strategy = *strategy

	switch event.Command {
	case "start":
		//TODO return if position is not 0

		// get pars from redis and declare strategy
		if err = s.commandStart(*session); err != nil {
			msg := fmt.Sprintf("Could not START strategy with key %s: %s", s.ch.RedisKey(*session), err)
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("Could not START strategy",
				zap.String("error", err.Error()),
				zap.String("command", "start"),
				zap.String("key", key))

		}
	case "stop":
		if err = s.commandStop(*session); err != nil {
			msg := fmt.Sprintf("Could not STOP strategy with key %s: %s", s.ch.RedisKey(*session), err)
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("Could not STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "stop"),
				zap.String("key", key))

		}
	case "hardStop":
		if err = s.commandHardStop(*session); err != nil {
			msg := fmt.Sprintf("Could not HARD_STOP strategy with key %s: %s", s.ch.RedisKey(*session), err)
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("Could not HARD_STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "hardStop"),
				zap.String("key", key))

		}
	case "softStop":
		if err = s.commandSoftStop(*session); err != nil {
			msg := fmt.Sprintf("Could not SOFT_STOP strategy with key %s: %s", s.ch.RedisKey(*session), err)
			s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
				UserId:  event.UserId,
				BotId:   event.BotId,
				Message: msg,
			})

			s.logger.Error("Could not SOFT_STOP strategy",
				zap.String("error", err.Error()),
				zap.String("command", "softStop"),
				zap.String("key", key))
		}
	default:
		msg := fmt.Sprintf("Event command not recognized: %s", event.Command)
		s.th.SendTelegramMessage(queue.MsgError, queue.RmqMessageEvent{
			UserId:  event.UserId,
			BotId:   event.BotId,
			Message: msg,
		})
	}
}

func (s *Strategy) commandStart(session terrabot.Session) (err error) {
	lastStatus := session.Strategy.Status

	session.Strategy.Status = terrabot.StatusStart
	if err := s.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", s.ch.RedisKey(session), err)
	}

	// if the status is soft stop, cancel it and put it to start
	if lastStatus == terrabot.StatusSoftStop {

		if err := s.ch.WriteStrategy(session); err != nil {
			return fmt.Errorf("could not store strategy with key %s: %s", s.ch.RedisKey(session), err)
		}

		return s.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStart)

	} else {

		// update position
		position := terrabot.Position{Symbol: session.Strategy.Symbol, PositionSide: session.Strategy.PositionSide, EntryPrice: 0, Size: 0, MarkPrice: 0}
		if err = s.ch.WritePosition(session, position); err != nil {
			return fmt.Errorf("could not store position in redis with key %s: %s", s.ch.RedisKey(session), err)
		}

		// update open orders
		if err = s.ch.WriteOpenOrders(session, terrabot.OpenOrders{}); err != nil {
			return fmt.Errorf("could not store open orders in redis with key %s: %s", s.ch.RedisKey(session), err)
		}

		err = s.StartStrategy(session)
		if err != nil {
			return err
		}

		return s.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStart)
	}
}

func (s *Strategy) commandStop(session terrabot.Session) error {
	session.Strategy.Status = terrabot.StatusStop

	if err := s.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", s.ch.RedisKey(session), err)
	}

	if err := s.cancelAllOpenOrders(session); err != nil {
		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	return s.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusStop)
}

func (s *Strategy) commandHardStop(session terrabot.Session) error {
	session.Strategy.Status = terrabot.StatusHardStop

	if err := s.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", s.ch.RedisKey(session), err)
	}

	if err := s.cancelAllOpenOrders(session); err != nil {
		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	if err := s.ClosePosition(session); err != nil {
		return fmt.Errorf("could not close position: %s", err)
	}

	err := s.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusHardStop)
	if err != nil {
		return err
	}

	s.ch.DeleteAllKeys(session)
	return nil
}

func (s *Strategy) commandSoftStop(session terrabot.Session) error {
	if session.Strategy.Status == terrabot.StatusStart {
		session.Strategy.Status = terrabot.StatusSoftStop
	} else {
		return fmt.Errorf("status is already %s", session.Strategy.Status)
	}

	key := s.ch.RedisKey(session)
	if err := s.ch.WriteStrategy(session); err != nil {
		return fmt.Errorf("could not store strategy with key %s: %s", key, err)
	}

	return s.dh.UpdateStrategyStatus(session.BotId, session.Strategy.Symbol, session.Strategy.PositionSide, terrabot.StatusSoftStop)
}

func (s *Strategy) cancelAllOpenOrders(session terrabot.Session) error {

	openOrders, err := s.getOpenOrders(session)
	if err != nil {
		return fmt.Errorf("could not cancel open orders - could not get open orders: %s", err)
	}

	if err := s.cancelMultipleOrders(session, openOrders); err != nil {

		s.logger.Error("Could not cancel open orders",
			zap.String("error", err.Error()),
			zap.String("key", s.ch.RedisKey(session)),
			zap.String("open orders list", fmt.Sprint(openOrders)),
		)

		return fmt.Errorf("could not cancel open orders: %s", err)
	}

	return nil
}
