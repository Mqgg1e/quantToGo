package logger

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	// Logger 全局日志实例
	Logger *zap.Logger
	// Sugar 全局Sugar日志实例（更易用）
	Sugar *zap.SugaredLogger
)

// Config 日志配置
type Config struct {
	Level      string // debug, info, warn, error
	OutputPath string // 日志文件路径
	MaxSize    int    // 单个文件最大大小（MB）
	MaxBackups int    // 保留的旧文件数量
	MaxAge     int    // 保留天数
	Compress   bool   // 是否压缩

	// 自动分文件配置
	SplitBySession bool   // 是否按会话分文件
	SessionID      string // 会话ID（时间戳）
}

// DefaultConfig 默认配置
func DefaultConfig() Config {
	return Config{
		Level:          "info",
		OutputPath:     "logs/trading.log",
		MaxSize:        100,   // 100MB
		MaxBackups:     10,    // 保留10个备份
		MaxAge:         30,    // 保留30天
		Compress:       true,  // 压缩旧文件
		SplitBySession: false, // 默认不分文件
		SessionID:      "",
	}
}

// Init 初始化日志系统
func Init(cfg Config) error {
	// 如果启用会话分文件，修改输出路径
	if cfg.SplitBySession && cfg.SessionID != "" {
		// 从原路径提取目录和扩展名
		dir := filepath.Dir(cfg.OutputPath)
		ext := filepath.Ext(cfg.OutputPath)
		base := filepath.Base(cfg.OutputPath)
		base = base[:len(base)-len(ext)]

		// 生成新的文件名：logs/trading_YYYYMMDD_HHMMSS.log
		cfg.OutputPath = filepath.Join(dir, fmt.Sprintf("%s_%s%s", base, cfg.SessionID, ext))
	}

	// 创建日志目录
	logDir := filepath.Dir(cfg.OutputPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return err
	}

	// 解析日志级别
	level := zapcore.InfoLevel
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 文件轮转配置
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.OutputPath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	// 控制台输出
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 创建核心
	core := zapcore.NewTee(
		// 文件输出：JSON格式
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			level,
		),
		// 控制台输出：彩色文本格式（更易读）
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			consoleWriter,
			level,
		),
	)

	// 创建Logger
	Logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Sugar = Logger.Sugar()

	return nil
}

// Close 关闭日志系统
func Close() {
	if Logger != nil {
		Logger.Sync()
	}
}

// 便捷方法

// Debug 调试日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}

// Info 信息日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Warn 警告日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Error 错误日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Fatal 致命错误日志
func Fatal(msg string, fields ...zap.Field) {
	Logger.Fatal(msg, fields...)
}

// WithFields 创建带字段的logger
func WithFields(fields ...zap.Field) *zap.Logger {
	return Logger.With(fields...)
}
