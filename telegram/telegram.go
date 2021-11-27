package telegram

import (
	"github.com/tbtc-bot/terrabot-sdk/queue"
	"go.uber.org/zap"
)

type TelegramHandler struct {
	Qh     *queue.QueueHandler
	Logger *zap.Logger
}

func (w *TelegramHandler) SendTelegramMessage(msgType queue.MessageType, body queue.RmqMessageEvent) {

	if err := w.Qh.PublishTelegramMessage(body, msgType); err != nil {
		w.Logger.Error(err.Error())
	}
}

func (w *TelegramHandler) SendTelegramTP(body queue.RmqTpEvent) {

	if err := w.Qh.PublishTelegramTp(body); err != nil {
		w.Logger.Error(err.Error())
	}
}
