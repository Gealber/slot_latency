package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Gealber/common/context"
	"github.com/Gealber/slot_latency/services"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Error loading .env file")
	}

	logFilePath := os.Getenv("LOG_FILE")
	if logFilePath == "" {
		logFilePath = "gossip.log"
	}

	logDir := filepath.Dir(logFilePath)
	if logDir != "." {
		err = os.MkdirAll(logDir, 0o755)
		if err != nil {
			log.Fatal().Err(err).Str("path", logDir).Msg("Failed to create log directory")
		}
	}

	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		log.Fatal().Err(err).Str("path", logFilePath).Msg("Failed to open log file")
	}
	defer func() {
		_ = logFile.Close()
	}()

	log.Logger = zerolog.New(io.MultiWriter(os.Stdout, logFile)).With().Timestamp().Logger()

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	logLevel := strings.ToLower(os.Getenv("LOG_LEVEL"))
	switch logLevel {
	case "trace":
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	log.Info().Str("log_file", logFilePath).Msg("Logger initialized")

	ctx, err := context.NewCtx(
		&services.TrackerService{},
	)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to init context")
		return
	}

	err = ctx.Run()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to fun")
		return
	}
}
