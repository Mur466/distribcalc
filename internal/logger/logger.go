package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"fmt"
)

var Logger *zap.Logger
var SLogger *zap.SugaredLogger

func InitLogger() {
	// Настройка конфигурации логгера
	config := zap.NewProductionConfig()

	// Уровень логирования
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	// Настройка логгера с конфигом
	var err error
	Logger, err = config.Build()
	if err != nil {
		fmt.Printf("Ошибка настройки логгера: %v\n", err)
	}
	SLogger =  Logger.Sugar()


}

