package server

import (
	"context"
	"fmt"
//	"game/http/server"

	"os"
	"os/signal"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Config struct {
	Width  int
	Height int
}

type Server struct {
	Cfg Config
}

func New(config Config) *Server {
	return &Server{
		Cfg: config,
	}
}

func (a *Server) Run(ctx context.Context) int {
	// Создание логгера с настройками для production
	logger := setupLogger()

	shutDownFunc, err := server.Run(ctx, logger, a.Cfg.Height, a.Cfg.Width)
	if err != nil {
		logger.Error(err.Error())

		return 1 // вернем код для регистрация ошибки системой
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	<-c
	cancel()
	//  завершим работу сервера
	shutDownFunc(ctx)

	return 0

}

// настройки логгера
func setupLogger() *zap.Logger {
	// Настройка конфигурации логгера
	config := zap.NewProductionConfig()

	// Уровень логирования
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	// Настройка логгера с конфигом
	logger, err := config.Build()
	if err != nil {
		fmt.Printf("Ошибка настройки логгера: %v\n", err)
	}

	return logger
}