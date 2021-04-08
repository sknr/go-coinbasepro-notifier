package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"runtime"
	"time"
)

func init() {
	// Define the standard logger
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).With().CallerWithSkipFrameCount(4).Logger()
}

func GetLogger() zerolog.Logger {
	return log.Logger
}

func LogDebug(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.DebugLevel, message)
		return
	}
	logLevel(zerolog.DebugLevel, message, data)
}

func LogDebugf(message string, values ...interface{}) {
	logLevel(zerolog.DebugLevel, fmt.Sprintf(message, values...))
}

func LogInfo(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.InfoLevel, message)
		return
	}
	logLevel(zerolog.InfoLevel, message, data)
}

func LogInfof(message string, values ...interface{}) {
	logLevel(zerolog.InfoLevel, fmt.Sprintf(message, values...))
}

func LogWarn(message string, data ...interface{}) {
	if len(data) == 0 {
		logLevel(zerolog.WarnLevel, message)
		return
	}
	logLevel(zerolog.WarnLevel, message, data)
}

func LogWarnf(message string, values ...interface{}) {
	logLevel(zerolog.WarnLevel, fmt.Sprintf(message, values...))
}

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

// DebugFileInfo returns the filename and number for debugging purposes
func DebugFileInfo(skip int) string {
	_, file, line, _ := runtime.Caller(skip)
	return fmt.Sprintf(" %s:%d", file, line)
}
