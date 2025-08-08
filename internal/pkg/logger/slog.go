package logger

import (
	"github.com/fandasy/06.08.2025/internal/models"
	"github.com/fandasy/06.08.2025/pkg/e"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"
)

func MustSet(env, dir string) *slog.Logger {
	log, err := Set(env, dir)
	if err != nil {
		panic(err)
	}

	return log
}

func Set(env, dir string) (*slog.Logger, error) {
	var output io.Writer
	if dir != "" {
		file, err := createFile(dir)
		if err != nil {
			return nil, err
		}

		output = file
	} else {
		output = os.Stdout
	}

	switch env {
	case models.EnvLocal:
		return slog.New(
			slog.NewTextHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug}),
		), nil
	case models.EnvDev:
		return slog.New(
			slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelDebug}),
		), nil
	case models.EnvProd:
		return slog.New(
			slog.NewJSONHandler(output, &slog.HandlerOptions{Level: slog.LevelInfo}),
		), nil
	default:
		return slog.Default(), nil
	}
}

func createFile(dir string) (*os.File, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0774); err != nil {
			return nil, e.Wrap("can't create a logs dir", err)
		}
	}

	nowDate := time.Now().Format(time.DateOnly)
	nowTime := strings.ReplaceAll(time.Now().Format(time.TimeOnly), ":", ".")

	file, err := os.Create(dir + "/" + nowDate + "_" + nowTime + ".log")
	if err != nil {
		return nil, e.Wrap("failed to create log file", err)
	}

	return file, nil
}
