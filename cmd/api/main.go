// cmd/api/main.go
package main

import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/rs/zerolog"
    "github.com/rs/zerolog/log"

    "url-shortener/internal/config"
)

func main() {
    // Pretty logging for dev
    log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

    cfg := config.Load()

    log.Info().Str("port", cfg.Server.Port).Msg("Starting URL shortener")

    srv := &http.Server{
        Addr:         fmt.Sprintf(":%s", cfg.Server.Port),
        ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
        WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
    }

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        log.Info().Msgf("Server listening on :%s", cfg.Server.Port)
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatal().Err(err).Msg("Server failed")
        }
    }()

    <-quit
    log.Info().Msg("Shutting down server...")

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Fatal().Err(err).Msg("Forced shutdown")
    }

    log.Info().Msg("Server stopped cleanly")
}