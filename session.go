package terrabot

import (
	"fmt"
)

type Session struct {
	BotId     string
	UserId    string
	ApiKey    string
	ApiSecret string
	Password  string
	Strategy  Strategy
}

func (s Session) String() string {
	return fmt.Sprintf("botId %s, userId %s, symbol %s, positionSide %s, status %s",
		s.BotId, s.UserId, s.Strategy.Symbol, s.Strategy.PositionSide, s.Strategy.Status)
}

func NewSession(botId string, userId, apiKey string, apiSecret string, strategy Strategy) *Session {
	return &Session{
		BotId:     botId,
		UserId:    userId,
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
		Strategy:  strategy,
	}
}