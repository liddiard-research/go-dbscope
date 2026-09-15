package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.SugaredLogger

func init() {
	config := zap.NewProductionConfig()

	config.Encoding = "console"
	config.Sampling = nil
	config.DisableStacktrace = false
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	Logger = logger.Sugar()
}
