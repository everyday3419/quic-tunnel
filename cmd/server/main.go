package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/everyday3419/quic-tunnel/internal/app"
	"github.com/everyday3419/quic-tunnel/internal/config"
	"github.com/rs/zerolog"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	cfg := config.NewDefaultConfig(":4242")
	srv, err := app.NewServer(cfg, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		if err := srv.Serve(ctx); err != nil {
			logger.Error().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown server")
	}
}
