package config

import (
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func InitLog() *zap.Logger {

	// configure Logging
	configLog := zap.NewProductionConfig()
	configLog.OutputPaths = []string{"stdout"}
	configLog.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
	configLog.DisableCaller = true
	configLog.EncoderConfig.LevelKey = "severity"
	configLog.EncoderConfig.MessageKey = "msg"
	logger, err := configLog.Build()

	// Not a good way - https://github.com/uber-go/zap/issues/717
	zap.ReplaceGlobals(logger)

	if err != nil {
		panic(err)
	}

	return logger

}
