package main

import (
	"log"

	"github.com/stpnv0/CommentTree/internal/app"
	"github.com/stpnv0/CommentTree/internal/config"
	wbflogger "github.com/wb-go/wbf/logger"
)

const appName = "CommentTree"

func main() {
	cfg := config.MustLoad()
	logger, err := wbflogger.InitLogger(
		cfg.Logger.LogEngine(),
		appName,
		cfg.Gin.Mode,
		wbflogger.WithLevel(cfg.Logger.LogLevel()),
	)
	if err != nil {
		log.Fatalf("init logger: %v", err)
	}

	application, err := app.New(cfg, logger)
	if err != nil {
		log.Fatalf("init app: %v", err)
	}

	if err = application.Run(); err != nil {
		log.Fatalf("app run: %v", err)
	}
}
