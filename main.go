package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/calindra/rollups-avail-reader/pkg/config"
	"github.com/calindra/rollups-avail-reader/pkg/paioavail"
	"github.com/calindra/rollups-base-reader/pkg/inputreader"
	"github.com/calindra/rollups-base-reader/pkg/paiodecoder"
	"github.com/calindra/rollups-base-reader/pkg/services"
	"github.com/calindra/rollups-base-reader/pkg/supervisor"
	"github.com/calindra/rollups-base-reader/pkg/transaction"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	_ "github.com/lib/pq"
	"github.com/lmittmann/tint"
)

func main() {
	var w supervisor.SupervisorWorker
	ctx := context.Background()
	startTime := time.Now()

	// Load configuration
	cfg := config.LoadConfig()

	// setup log
	logOpts := new(tint.Options)
	logOpts.Level = cfg.LogLevel
	logOpts.AddSource = cfg.LogAddSource
	logOpts.NoColor = !cfg.LogUseColor
	logOpts.TimeFormat = cfg.LogTimeFormat
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      cfg.ServerTimeout,
	}))

	paio := paioavail.NewPaioSender2Server("")
	sender := transaction.TransactionAPI{
		ClientSender: paio,
	}
	transaction.Register(e, &sender)

	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort),
		Handler: e,
	})

	db := CreateDBInstance(cfg)
	if db == nil {
		log.Fatal("Failed to create database instance")
	}

	inputService := services.NewInputService(db)

	paioPath, err := paiodecoder.DownloadPaioDecoderExecutableAsNeeded()
	if err != nil {
		log.Fatal("Failed to download paio decoder executable:", err)
	}

	inputReaderWorker := &inputreader.InputReaderWorker{
		Provider: cfg.BlockchainWsEndpoint,
	}

	availFromBlock := cfg.StartBlock - cfg.BlockOffset

	listener := paioavail.NewAvailListener(availFromBlock, inputService, inputReaderWorker, 0, paioPath)

	w.Workers = append(w.Workers, listener)

	ready := make(chan struct{}, 1)
	go func() {
		select {
		case <-ready:
			slog.Info("Started the server")
			slog.Info("cartesi-rollups-graphql: ready", "after", time.Since(startTime))
		case <-ctx.Done():
		}
	}()

	slog.Info("I'll start the server")

	// Start the supervisor worker
	if err := w.Start(ctx, ready); err != nil {
		slog.Error("Error starting supervisor worker:", "error", err)
		return
	}
}

func CreateDBInstance(cfg *config.Config) *sqlx.DB {
	var db *sqlx.DB
	slog.Info("Using PostGres DB ...")

	db = sqlx.MustConnect("postgres", cfg.DatabaseConnection)
	configureConnectionPool(db, cfg)
	return db
}

func configureConnectionPool(db *sqlx.DB, cfg *config.Config) {
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
}