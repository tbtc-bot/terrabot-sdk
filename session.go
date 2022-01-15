package terrabot

import (
	"fmt"
)

type Session struct {
	Exchange  string
	BotId     string
	UserId    string
	ApiKey    string
	ApiSecret string
	Password  string
	Strategy  *Strategy
	Simulated bool
}

func (s Session) String() string {
	return fmt.Sprintf("exchange %s, botId %s, userId %s, symbol %s, positionSide %s, status %s",
		s.Exchange, s.BotId, s.UserId, s.Strategy.Symbol, s.Strategy.PositionSide, s.Strategy.Status)
}

func NewSession(exchange string, botId string, userId, apiKey string, apiSecret string, password string, simulated bool) *Session {
	return &Session{
		Exchange:  exchange,
		BotId:     botId,
		UserId:    userId,
		ApiKey:    apiKey,
		ApiSecret: apiSecret,
		Password:  password,
		Strategy:  &Strategy{},
		Simulated: simulated,
	}
}
