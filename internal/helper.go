package internal

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/shopspring/decimal"
	"github.com/sknr/go-coinbasepro-notifier/internal/logger"
	"os"
)

// HasError returns true if an error exists
func HasError(err error) bool {
	return err != nil
}


// PanicOnError checks panics if an error exists and does nothing otherwise
func PanicOnError(err error) {
	if HasError(err) {
		panic(err)
	}
}

// StringToDecimal converts an string into a Decimal value or 0 in case of an error
func StringToDecimal(number string) decimal.Decimal {
	if number == "" {
		return decimal.NewFromInt(0)
	}
	result, err := decimal.NewFromString(number)
	if err != nil {
		logger.LogError(err)
		result = decimal.NewFromInt(0)
	}

	return result
}

// CheckEnvVars loads environment variables from .env file if exists
func CheckEnvVars(envVars ...string) {
	err := godotenv.Load()
	if err == nil {
		logger.LogInfo(".env file found => using values from .env file instead of OS env vars")
	}

	for _, ev := range envVars {
		if _, ok := os.LookupEnv(ev); ok == false {
			panic(fmt.Sprintf("Required env var %s is missing", ev))
		}
	}
}
