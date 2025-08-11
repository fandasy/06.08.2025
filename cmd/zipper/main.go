package main

import (
	"context"
	"flag"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fandasy/06.08.2025/internal/app"
	"github.com/fandasy/06.08.2025/internal/config"
	"github.com/fandasy/06.08.2025/internal/models"
	"github.com/fandasy/06.08.2025/internal/pkg/logger"
	"github.com/fandasy/06.08.2025/internal/pkg/logger/sl"
)

func main() {
	attr := getRunningAttr()

	cfg := config.MustLoad(attr.config_path)

	log := logger.MustSet(attr.env, cfg.Logger.Dir)

	application := app.MustNew(attr.env, cfg, log)

	go application.MustRun(log)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	sign := <-stop
	log.Info("Signal intercepted, application stops", slog.String("signal", sign.String()))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := application.Shutdown(ctx, log); err != nil {
		log.Error("Application shutdown error", sl.Err(err))
	}
}

type Attr struct {
	env         string
	config_path string
}

func getRunningAttr() Attr {
	const (
		defaultConfigPath = "./config/local.yaml"
		defaultEnv        = models.EnvLocal
	)

	var env string
	flag.StringVar(&env,
		"env",
		"",
		"environment",
	)

	var config_path string
	flag.StringVar(&config_path,
		"config",
		"",
		"config file path",
	)

	flag.Parse()

	if env == "" {
		env = os.Getenv("ENV")
		if env == "" {
			env = defaultEnv
		}
	}

	if config_path == "" {
		config_path = os.Getenv("CONFIG_PATH")
		if config_path == "" {
			config_path = defaultConfigPath
		}
	}

	return Attr{
		env:         env,
		config_path: config_path,
	}
}
