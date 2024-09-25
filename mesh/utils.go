package mesh

import (
	"log/slog"
	"time"
)

func logInfo(requestID string, msg string) {
	slog.Info(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

// TODO: Combine w/ logInfo and dynamically call appropriate log level func
func logError(requestID string, msg string) {
	slog.Error(msg, "source", "heroku-integration-service-mesh", "request-id", requestID)
}

func timeTrack(requestID string, startTime time.Time, name string) {
	elapsedTime := time.Since(startTime)
	logInfo(requestID, name+" took "+elapsedTime.String())
}
