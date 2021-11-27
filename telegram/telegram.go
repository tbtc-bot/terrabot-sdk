package telegram

import (
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"go.uber.org/zap"
)

type TelegramHandler struct {
	Qh     *queue.QueueHandler
	Logger *zap.Logger
}

func (th *TelegramHandler) SendTelegramMessage(msgType queue.MessageType, body queue.RmqMessageEvent) {

	if err := th.Qh.PublishTelegramMessage(body, msgType); err != nil {
		th.Logger.Error(err.Error())
	}
}

func (th *TelegramHandler) SendTelegramTP(body queue.RmqTpEvent) {

	if err := th.Qh.PublishTelegramTp(body); err != nil {
		th.Logger.Error(err.Error())
	}
}
