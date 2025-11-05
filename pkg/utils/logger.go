package utils

import (
	"log/slog"
	"os"
	"path/filepath"
)

var logger *slog.Logger

// InitLogger 初始化日志系统
func InitLogger() error {
	// 获取可执行文件路径
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	// 获取可执行文件所在目录
	exeDir := filepath.Dir(exePath)

	// 创建日志文件路径
	logPath := filepath.Join(exeDir, "genshin-starcraft-mcp.log")

	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return err
	}

	// 根据DEBUG环境变量设置日志级别
	level := slog.LevelInfo
	if os.Getenv("DEBUG") == "true" {
		level = slog.LevelDebug
	}

	// 设置slog输出到文件
	handler := slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: level,
	})
	logger = slog.New(handler)

	return nil
}

// GetLogger 获取日志实例
func GetLogger() *slog.Logger {
	if logger == nil {
		// 如果未初始化，创建一个默认的stderr日志器
		handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelWarn,
		})
		logger = slog.New(handler)
	}
	return logger
}

// Debug 调试日志
func Debug(msg string, args ...interface{}) {
	if os.Getenv("DEBUG") == "true" {
		GetLogger().Debug(msg, args...)
	}
}

// Info 信息日志
func Info(msg string, args ...interface{}) {
	GetLogger().Info(msg, args...)
}

// Error 错误日志
func Error(msg string, args ...interface{}) {
	GetLogger().Error(msg, args...)
}