package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/calindra/rollups-avail-reader/pkg/paioavail"
	"github.com/calindra/rollups-base-reader/pkg/inputreader"
	"github.com/calindra/rollups-base-reader/pkg/paiodecoder"
	"github.com/calindra/rollups-base-reader/pkg/repository"
	"github.com/calindra/rollups-base-reader/pkg/services"
	"github.com/calindra/rollups-base-reader/pkg/supervisor"
	"github.com/calindra/rollups-base-reader/pkg/transaction"
	"github.com/jmoiron/sqlx"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/lmittmann/tint"
	"github.com/spf13/cast"
)

const timeout = 30 * time.Second

func main() {
	var w supervisor.SupervisorWorker
	ctx := context.Background()
	startTime := time.Now()
	rpcURL := os.Getenv("CARTESI_BLOCKCHAIN_WS_ENDPOINT")
	if rpcURL == "" {
		log.Fatal("CARTESI_BLOCKCHAIN_WS_ENDPOINT not set")
	}

	// setup log
	logOpts := new(tint.Options)
	logOpts.Level = slog.LevelDebug
	logOpts.AddSource = true
	logOpts.NoColor = false
	logOpts.TimeFormat = "[15:04:05.000]"
	handler := tint.NewHandler(os.Stdout, logOpts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	e := echo.New()
	e.Use(middleware.CORS())
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		ErrorMessage: "Request timed out",
		Timeout:      timeout,
	}))

	paio := paioavail.NewPaioSender2Server("")
	sender := transaction.TransactionAPI{
		ClientSender: paio,
	}
	transaction.Register(e, &sender)

	w.Workers = append(w.Workers, supervisor.HttpWorker{
		Address: fmt.Sprintf("%s:%d", "0.0.0.0", 8080),
		Handler: e,
	})

	db := CreateDBInstance()
	if db == nil {
		log.Fatal("Failed to create database instance")
	}

	inputRepository := repository.NewInputRepository(db)
	epochRepository := repository.NewEpochRepository(db)
	appRepository := repository.NewAppRepository(db)
	inputService := services.NewInputService(inputRepository, epochRepository, appRepository)

	paioPath, err := paiodecoder.DownloadPaioDecoderExecutableAsNeeded()

	if err != nil {
		log.Fatal("Failed to download paio decoder executable:", err)
	}

	inputReaderWorker := &inputreader.InputReaderWorker{
		Provider: rpcURL,
	}

	availFromBlock := uint64(1630753 - 25)

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

// Refactor in the back

const (
	DefaultHttpPort           = 8080
	DefaultRollupsPort        = 5004
	DefaultNamespace          = 10008
	DefaultMaxOpenConnections = 25
	DefaultMaxIdleConnections = 10
	DefaultConnMaxLifetime    = 30 * time.Minute
	DefaultConnMaxIdleTime    = 5 * time.Minute
)

func CreateDBInstance() *sqlx.DB {
	var db *sqlx.DB
	slog.Info("Using PostGres DB ...")
	dbUrl, ok := os.LookupEnv("CARTESI_DATABASE_CONNECTION")
	if !ok {
		log.Fatal("CARTESI_DATABASE_CONNECTION not set")
	}
	db = sqlx.MustConnect("postgres", dbUrl)
	configureConnectionPool(db)
	return db
}

// configureConnectionPool sets the connection pool settings for the database connection.
// The following environment variables are used to configure the connection pool:
// - DB_MAX_OPEN_CONNS: Maximum number of open connections to the database
// - DB_MAX_IDLE_CONNS: Maximum number of idle connections in the pool
// - DB_CONN_MAX_LIFETIME: Maximum amount of time a connection may be reused
// - DB_CONN_MAX_IDLE_TIME: Maximum amount of time a connection may be idle
func configureConnectionPool(db *sqlx.DB) {
	defaultConnMaxLifetime := int(DefaultConnMaxLifetime.Seconds())
	defaultConnMaxIdleTime := int(DefaultConnMaxIdleTime.Seconds())

	maxOpenConns := getEnvInt("DB_MAX_OPEN_CONNS", DefaultMaxOpenConnections)
	maxIdleConns := getEnvInt("DB_MAX_IDLE_CONNS", DefaultMaxIdleConnections)
	connMaxLifetime := getEnvInt("DB_CONN_MAX_LIFETIME", defaultConnMaxLifetime)
	connMaxIdleTime := getEnvInt("DB_CONN_MAX_IDLE_TIME", defaultConnMaxIdleTime)
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)
	db.SetConnMaxLifetime(time.Duration(connMaxLifetime) * time.Second)
	db.SetConnMaxIdleTime(time.Duration(connMaxIdleTime) * time.Second)
}

func getEnvInt(envName string, defaultValue int) int {
	value, exists := os.LookupEnv(envName)
	if !exists {
		return defaultValue
	}
	intValue, err := cast.ToIntE(value)
	if err != nil {
		slog.Error("configuration error", "envName", envName, "value", value)
		panic(err)
	}
	return intValue
}
