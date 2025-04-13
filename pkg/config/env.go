package config

import (
	"log"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cast"
)

type Config struct {
	// Server Configuration
	ServerHost      string
	ServerPort      int
	ServerTimeout   time.Duration
	RollupsPort    int
	Namespace      int

	// Blockchain Configuration
	BlockchainWsEndpoint string
	StartBlock          uint64
	BlockOffset         uint64

	// Database Configuration
	DatabaseConnection  string
	MaxOpenConns       int
	MaxIdleConns       int
	ConnMaxLifetime    time.Duration
	ConnMaxIdleTime    time.Duration

	// Logging Configuration
	LogLevel      slog.Level
	LogAddSource  bool
	LogUseColor   bool
	LogTimeFormat string
}

func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntWithDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := cast.ToIntE(value)
		if err != nil {
			slog.Error("configuration error", "key", key, "value", value)
			panic(err)
		}
		return intValue
	}
	return defaultValue
}

func getEnvBoolWithDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := cast.ToBoolE(value)
		if err != nil {
			slog.Error("configuration error", "key", key, "value", value)
			panic(err)
		}
		return boolValue
	}
	return defaultValue
}

func LoadConfig() *Config {
	// Default values
	config := &Config{
		// Server defaults
		ServerHost:    getEnvWithDefault("AVAIL_READER_SERVER_HOST", "0.0.0.0"),
		ServerPort:    getEnvIntWithDefault("AVAIL_READER_SERVER_PORT", 8080),
		ServerTimeout: time.Duration(getEnvIntWithDefault("AVAIL_READER_SERVER_TIMEOUT", 30)) * time.Second,
		RollupsPort:   getEnvIntWithDefault("AVAIL_READER_ROLLUPS_PORT", 5004),
		Namespace:     getEnvIntWithDefault("AVAIL_READER_NAMESPACE", 10008),

		// Blockchain defaults
		BlockchainWsEndpoint: os.Getenv("CARTESI_BLOCKCHAIN_WS_ENDPOINT"), // Required
		StartBlock:          uint64(getEnvIntWithDefault("AVAIL_READER_START_BLOCK", 1630728)),
		BlockOffset:         uint64(getEnvIntWithDefault("AVAIL_READER_BLOCK_OFFSET", 25)),

		// Database defaults
		DatabaseConnection: os.Getenv("CARTESI_DATABASE_CONNECTION"), // Required
		MaxOpenConns:      getEnvIntWithDefault("DB_MAX_OPEN_CONNS", 25),
		MaxIdleConns:      getEnvIntWithDefault("DB_MAX_IDLE_CONNS", 10),
		ConnMaxLifetime:   time.Duration(getEnvIntWithDefault("DB_CONN_MAX_LIFETIME", 1800)) * time.Second, // 30 minutes
		ConnMaxIdleTime:   time.Duration(getEnvIntWithDefault("DB_CONN_MAX_IDLE_TIME", 300)) * time.Second, // 5 minutes

		// Logging defaults
		LogLevel:      slog.LevelDebug, // TODO: Add proper level parsing from env
		LogAddSource:  getEnvBoolWithDefault("AVAIL_READER_LOG_ADD_SOURCE", true),
		LogUseColor:   getEnvBoolWithDefault("AVAIL_READER_LOG_USE_COLOR", true),
		LogTimeFormat: getEnvWithDefault("AVAIL_READER_LOG_TIME_FORMAT", "[15:04:05.000]"),
	}

	// Validate required environment variables
	if config.BlockchainWsEndpoint == "" {
		log.Fatal("CARTESI_BLOCKCHAIN_WS_ENDPOINT not set")
	}
	if config.DatabaseConnection == "" {
		log.Fatal("CARTESI_DATABASE_CONNECTION not set")
	}

	return config
}