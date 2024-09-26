package mesh

import (
	"log/slog"
	"time"
)

const (
	HdrNameRequestID = "x-request-id"
)

func logInfo(requestID string, msg string) {
	slog.Info(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

// TODO: Combine w/ logInfo and dynamically call appropriate log level func
func logDebug(requestID string, msg string) {
	slog.Debug(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

// TODO: Combine w/ logInfo and dynamically call appropriate log level func
func logWarn(requestID string, msg string) {
	slog.Warn(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

// TODO: Combine w/ logInfo and dynamically call appropriate log level func
func logError(requestID string, msg string) {
	slog.Error(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

func timeTrack(requestID string, startTime time.Time, name string) {
	elapsedTime := time.Since(startTime)
	logInfo(requestID, name+" took "+elapsedTime.String())
}
