package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

func InitLogger(logFile string, maxSize, maxBackups, maxAge int) {
	fileLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    maxSize, // megabytes
		MaxBackups: maxBackups,
		MaxAge:     maxAge, // days
		Compress:   true,
	}

	// Write to both file and stdout
	multiWriter := io.MultiWriter(os.Stdout, fileLogger)

	handler := slog.NewJSONHandler(multiWriter, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})

	logger := slog.New(handler)
	slog.SetDefault(logger)
}
