package config_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ImSoHappi/test-haulmer/payments-api/internal/config"
)

func setValidEnv() func() {
	os.Setenv("PORT", "8081")
	os.Setenv("DATABASE_URL", "postgres://user:pass@localhost:5432/payments?sslmode=disable")
	os.Setenv("ACQUIRER_BASE_URL", "http://custom-mock:9000")
	os.Setenv("PAYMENT_MAX_AMOUNT_CLP", "5000000")
	os.Setenv("LOG_LEVEL", "DEBUG")

	return func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DATABASE_URL")
		os.Unsetenv("ACQUIRER_BASE_URL")
		os.Unsetenv("PAYMENT_MAX_AMOUNT_CLP")
		os.Unsetenv("LOG_LEVEL")
	}
}

func TestLoad_Success_WithPort(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()

	cfg, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "8081", cfg.Port)
	assert.Equal(t, "postgres://user:pass@localhost:5432/payments?sslmode=disable", cfg.DatabaseURL)
	assert.Equal(t, "http://custom-mock:9000", cfg.AcquirerBaseURL)
	assert.Equal(t, int64(5000000), cfg.PaymentMaxAmountCLP)
	assert.Equal(t, "DEBUG", cfg.LogLevel)
}

func TestLoad_FailFast_MissingPort(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Unsetenv("PORT")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "PORT is required")
}

func TestLoad_FailFast_MissingDatabaseURL(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Unsetenv("DATABASE_URL")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "DATABASE_URL is required")
}

func TestLoad_FailFast_MissingAcquirerBaseURL(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Unsetenv("ACQUIRER_BASE_URL")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "ACQUIRER_BASE_URL is required")
}

func TestLoad_FailFast_MissingMaxAmount(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Unsetenv("PAYMENT_MAX_AMOUNT_CLP")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "PAYMENT_MAX_AMOUNT_CLP is required")
}

func TestLoad_FailFast_InvalidMaxAmount(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Setenv("PAYMENT_MAX_AMOUNT_CLP", "invalid-number")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "PAYMENT_MAX_AMOUNT_CLP")
}

func TestLoad_FailFast_MissingLogLevel(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Unsetenv("LOG_LEVEL")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "LOG_LEVEL is required")
}

func TestLoad_FailFast_InvalidLogLevel(t *testing.T) {
	cleanup := setValidEnv()
	defer cleanup()
	os.Setenv("LOG_LEVEL", "VERBOSE")

	cfg, err := config.Load()
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "LOG_LEVEL invalid")
}
