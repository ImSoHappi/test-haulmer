package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration loaded strictly from environment variables.
type Config struct {
	Port                string
	DatabaseURL         string
	AcquirerBaseURL     string
	PaymentMaxAmountCLP int64
	LogLevel            string
}

// Load parses environment variables and validates critical required parameters.
// Fails fast if any mandatory variable (PORT, DATABASE_URL, ACQUIRER_BASE_URL, PAYMENT_MAX_AMOUNT_CLP, LOG_LEVEL) is missing.
func Load() (*Config, error) {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		return nil, errors.New("PORT is required")
	}

	dbURL := strings.TrimSpace(os.Getenv("DATABASE_URL"))
	if dbURL == "" {
		return nil, errors.New("DATABASE_URL is required")
	}

	acquirerURL := strings.TrimSpace(os.Getenv("ACQUIRER_BASE_URL"))
	if acquirerURL == "" {
		return nil, errors.New("ACQUIRER_BASE_URL is required")
	}

	maxAmountStr := strings.TrimSpace(os.Getenv("PAYMENT_MAX_AMOUNT_CLP"))
	if maxAmountStr == "" {
		return nil, errors.New("PAYMENT_MAX_AMOUNT_CLP is required")
	}
	maxAmount, err := strconv.ParseInt(maxAmountStr, 10, 64)
	if err != nil || maxAmount <= 0 {
		return nil, fmt.Errorf("PAYMENT_MAX_AMOUNT_CLP must be a positive integer, got %q: %w", maxAmountStr, err)
	}

	logLevel := strings.ToUpper(strings.TrimSpace(os.Getenv("LOG_LEVEL")))
	if logLevel == "" {
		return nil, errors.New("LOG_LEVEL is required")
	}
	switch logLevel {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return nil, fmt.Errorf("LOG_LEVEL invalid %q, expected DEBUG, INFO, WARN or ERROR", logLevel)
	}

	return &Config{
		Port:                port,
		DatabaseURL:         dbURL,
		AcquirerBaseURL:     acquirerURL,
		PaymentMaxAmountCLP: maxAmount,
		LogLevel:            logLevel,
	}, nil
}
