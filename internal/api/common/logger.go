package common

import (
	"time"

	"gofr.dev/pkg/gofr/logging"
)

// LogRequest uses GoFr's logging.Logger interface to record request details.
func LogRequest(logger logging.Logger, method, path string, duration time.Duration) {
	logger.Infof("Method: %s | Path: %s | Duration: %v", method, path, duration)
}
