package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"time"
)

func init() {
	// Define the standard logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().CallerWithSkipFrameCount(4).Logger()
}

// GetLogger returns the current zerolog.Logger instance
func GetLogger() zerolog.Logger {
	return log.Logger
}

// LogDebug logs a debug message and optional additional data
func LogDebug(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.DebugLevel, message)
		return
	}
	logLevel(zerolog.DebugLevel, message, data)
}

// LogDebugf logs a debug message which can be formatted as fmt.Sprintf
func LogDebugf(message string, values ...interface{}) {
	logLevel(zerolog.DebugLevel, fmt.Sprintf(message, values...))
}

// LogInfo logs an info message and optional additional data
func LogInfo(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.InfoLevel, message)
		return
	}
	logLevel(zerolog.InfoLevel, message, data)
}

// LogInfof logs an info message which can be formatted as fmt.Sprintf
func LogInfof(message string, values ...interface{}) {
	logLevel(zerolog.InfoLevel, fmt.Sprintf(message, values...))
}

// LogWarn logs a warning message and optional additional data
func LogWarn(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.WarnLevel, message)
		return
	}
	logLevel(zerolog.WarnLevel, message, data)
}

// LogWarnf logs a warning message which can be formatted as fmt.Sprintf
func LogWarnf(message string, values ...interface{}) {
	logLevel(zerolog.WarnLevel, fmt.Sprintf(message, values...))
}

// LogError logs an error
func LogError(err error, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.ErrorLevel, err.Error())
		return
	}
	logLevel(zerolog.ErrorLevel, err.Error(), data)
}

// LogErrorIfExists logs an error if it exits
func LogErrorIfExists(err error, data ...interface{}) {
	if err != nil {
		if len(data) == 0 {
			logLevel(zerolog.ErrorLevel, err.Error())
			return
		}
		logLevel(zerolog.ErrorLevel, err.Error(), data)
	}
}

func logLevel(level zerolog.Level, message string, data ...interface{}) {
	if len(data) == 0 {
		log.WithLevel(level).Msg(message)
		return
	}
	log.WithLevel(level).Interface("data", data).Msg(message)
}
