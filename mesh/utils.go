package mesh

import (
	"log/slog"
	"time"
)

const (
	HdrNameRequestID = "x-request-id"
)

func LogInfo(requestID string, msg string) {
	slog.Info(msg, "request-id", requestID)
}

// TODO: Combine w/ LogInfo and dynamically call appropriate log level func
func LogDebug(requestID string, msg string) {
	slog.Debug(msg, "request-id", requestID)
}

// TODO: Combine w/ LogInfo and dynamically call appropriate log level func
func LogWarn(requestID string, msg string) {
	slog.Warn(msg, "request-id", requestID)
}

// TODO: Combine w/ LogInfo and dynamically call appropriate log level func
func LogError(requestID string, msg string) {
	slog.Error(msg, "request-id", requestID)
}

func TimeTrack(requestID string, startTime time.Time, name string) {
	elapsedTime := time.Since(startTime)
	LogDebug(requestID, name+" took "+elapsedTime.String())
}
