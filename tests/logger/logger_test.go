package logger_test

import (
	"bytes"
	"strings"
	"testing"

	"pogonian/internal/logger"
)

func TestLogger_StructuredLogging(t *testing.T) {
	// 1. Test Console text mode
	var consoleBuf bytes.Buffer
	l := logger.NewLogger(&consoleBuf, logger.LevelInfo, false)
	l.Info("Server started on port 3000", map[string]any{"port": 3000, "host": "127.0.0.1"})

	consoleOut := consoleBuf.String()
	if !strings.Contains(consoleOut, "[INFO]") || !strings.Contains(consoleOut, "Server started on port 3000") {
		t.Errorf("Expected console formatted log, got:\n%s", consoleOut)
	}

	// 2. Test JSON mode
	var jsonBuf bytes.Buffer
	jl := logger.NewLogger(&jsonBuf, logger.LevelDebug, true)
	jl.Error("Database connection lost", map[string]any{"db": "sqlite", "code": 500})

	jsonOut := jsonBuf.String()
	if !strings.Contains(jsonOut, `"level":"ERROR"`) || !strings.Contains(jsonOut, `"message":"Database connection lost"`) {
		t.Errorf("Expected JSON structured log, got:\n%s", jsonOut)
	}
}
