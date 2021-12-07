package strategy

import (
	"fmt"
	"strconv"

	"github.com/tbtc-bot/terrabot-sdk"
	"github.com/tbtc-bot/terrabot-sdk/cache"
	"go.uber.org/zap"
)

func (sh *Strategy) walletBalanceFromAPI(session terrabot.Session) (terrabot.WalletBalance, error) {
	balance, err := sh.eh.GetBalanceRetry(session, ATTEMPTS, SLEEP)
	if err != nil {
		return nil, fmt.Errorf("could not get wallet balance from binance: %s", err)
	}

	err = sh.ch.WriteWalletBalance(session, balance)
	if err != nil {
		sh.logger.Error("Could not store balance in redis",
			zap.String("botId", session.BotId),
			zap.String("key", cache.GetRedisKeyWalletBalance(sh.ch.Exchange, session.BotId)),
			zap.String("error", err.Error()),
		)
	}
	return balance, nil
}

func (sh *Strategy) getAssetBalance(session terrabot.Session, asset terrabot.Asset) (float64, error) {
	walletBalance, err := sh.walletBalanceFromAPI(session)
	if err != nil {

		// if api fails, get from redis
		walletBalance, err = sh.ch.ReadWalletBalance(session.BotId)
		if err != nil {
			return 0, err
		}
	}

	for _, b := range walletBalance {
		if terrabot.Asset(b.Asset) == asset {
			bal, err := strconv.ParseFloat(b.Balance, 64)
			if err != nil {
				return 0, fmt.Errorf("could not parse string %s to float", b.AvailableBalance)
			}
			return bal, nil
		}
	}

	return 0, fmt.Errorf("asset %s not found in wallet balance", asset)
}
