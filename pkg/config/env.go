package config

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/spf13/cast"
)

const (
	// Environment variables
	EnvServerHost    = "AVAIL_READER_SERVER_HOST"
	EnvServerPort    = "AVAIL_READER_SERVER_PORT"
	EnvServerTimeout = "AVAIL_READER_SERVER_TIMEOUT"
	EnvRollupsPort   = "AVAIL_READER_ROLLUPS_PORT"
	EnvNamespace     = "AVAIL_READER_NAMESPACE"
	EnvStartBlock    = "AVAIL_READER_START_BLOCK"
	EnvBlockOffset   = "AVAIL_READER_BLOCK_OFFSET"
	EnvLogLevel      = "AVAIL_READER_LOG_LEVEL"
	EnvLogAddSource  = "AVAIL_READER_LOG_ADD_SOURCE"
	EnvLogUseColor   = "AVAIL_READER_LOG_USE_COLOR"
	EnvLogTimeFormat = "AVAIL_READER_LOG_TIME_FORMAT"

	// TODO add prefix AVAIL_READER_ to the legacy env vars
	EnvBlockchainWsEndpoint = "CARTESI_BLOCKCHAIN_WS_ENDPOINT"
	EnvDatabaseConnection   = "CARTESI_DATABASE_CONNECTION"
	EnvMaxOpenConns         = "DB_MAX_OPEN_CONNS"
	EnvMaxIdleConns         = "DB_MAX_IDLE_CONNS"
	EnvConnMaxLifetime      = "DB_CONN_MAX_LIFETIME"
	EnvConnMaxIdleTime      = "DB_CONN_MAX_IDLE_TIME"

	// Others
	DefaultLogLevel = "DEBUG"
)

type Config struct {
	// Server Configuration
	ServerHost    string
	ServerPort    int
	ServerTimeout time.Duration
	RollupsPort   int
	Namespace     int

	// Blockchain Configuration
	BlockchainWsEndpoint string
	StartBlock           uint64
	BlockOffset          uint64

	// Database Configuration
	DatabaseConnection string
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

// converts a string into slog.Level object.
func parseLogLevel(level string) (slog.Level, error) {
	switch strings.ToUpper(level) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid log level: %s. Valid values are: DEBUG, INFO, WARN, ERROR", level)
	}
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
			slog.Error("configuration error: Not Int", "key", key, "value", value)
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
			slog.Error("configuration error: Not Bool", "key", key, "value", value)
			panic(err)
		}
		return boolValue
	}
	return defaultValue
}

func getLogLevel() slog.Level {
	levelStr := getEnvWithDefault(EnvLogLevel, DefaultLogLevel)
	level, err := parseLogLevel(levelStr)
	if err != nil {
		slog.Warn("invalid log level, using default",
			"error", err,
			"provided", levelStr,
			"default", DefaultLogLevel)
		return slog.LevelDebug
	}
	return level
}

func LoadConfig() *Config {
	// Default values
	config := &Config{
		// Server defaults
		ServerHost:    getEnvWithDefault(EnvServerHost, "0.0.0.0"),
		ServerPort:    getEnvIntWithDefault(EnvServerPort, 8080),
		ServerTimeout: time.Duration(getEnvIntWithDefault(EnvServerTimeout, 30)) * time.Second,
		RollupsPort:   getEnvIntWithDefault(EnvRollupsPort, 5004),
		Namespace:     getEnvIntWithDefault(EnvNamespace, 10008),

		// Blockchain defaults
		BlockchainWsEndpoint: os.Getenv(EnvBlockchainWsEndpoint), // Required
		StartBlock:           uint64(getEnvIntWithDefault(EnvStartBlock, 1630728)),
		BlockOffset:          uint64(getEnvIntWithDefault(EnvBlockOffset, 25)),

		// Database defaults
		DatabaseConnection: os.Getenv(EnvDatabaseConnection), // Required
		MaxOpenConns:       getEnvIntWithDefault(EnvMaxOpenConns, 25),
		MaxIdleConns:       getEnvIntWithDefault(EnvMaxIdleConns, 10),
		ConnMaxLifetime:    time.Duration(getEnvIntWithDefault(EnvConnMaxLifetime, 1800)) * time.Second, // 30 minutes
		ConnMaxIdleTime:    time.Duration(getEnvIntWithDefault(EnvConnMaxIdleTime, 300)) * time.Second,  // 5 minutes

		// Logging defaults
		LogLevel:      getLogLevel(),
		LogAddSource:  getEnvBoolWithDefault(EnvLogAddSource, true),
		LogUseColor:   getEnvBoolWithDefault(EnvLogUseColor, true),
		LogTimeFormat: getEnvWithDefault(EnvLogTimeFormat, "[15:04:05.000]"),
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
