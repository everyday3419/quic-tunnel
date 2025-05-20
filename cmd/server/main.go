package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/everyday3419/quic-tunnel/internal/config"
	"github.com/everyday3419/quic-tunnel/internal/server"
	"github.com/rs/zerolog"
)

func main() {
	conf, err := config.New[*config.ServerConfig, config.ServerConfigOpts]("conf/server.toml")
	if err != nil {
		log.Fatal(err)
	}

	output := zerolog.ConsoleWriter{Out: os.Stdout, NoColor: false}
	logger := zerolog.New(output).With().Timestamp().Logger()

	srv, err := server.New(conf.Addr, conf.TLSConfig, conf.QUICConfig, &logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("failed to create server")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	go func() {
		if err := srv.Run(ctx); err != nil {
			logger.Error().Err(err).Msg("server failed")
		}
	}()

	<-ctx.Done()
	if err := srv.Shutdown(); err != nil {
		logger.Error().Err(err).Msg("failed to shutdown server")
	}
}
