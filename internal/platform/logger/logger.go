package logger

import (
	"go.uber.org/zap"
)

var Log *zap.SugaredLogger

func Init(dev bool) {
	var logger *zap.Logger
	var err error

	if dev {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(err)
	}

	Log = logger.Sugar()
}
