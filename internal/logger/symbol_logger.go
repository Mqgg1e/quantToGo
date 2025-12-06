package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// SymbolLogger 按品种分文件的日志记录器
type SymbolLogger struct {
	sessionID  string                 // 会话ID（启动时间）
	baseDir    string                 // 基础目录
	loggers    map[string]*zap.Logger // symbol -> logger
	mu         sync.RWMutex
	level      zapcore.Level
	maxSize    int
	maxBackups int
	maxAge     int
	compress   bool
}

// NewSymbolLogger 创建按品种分文件的日志记录器
func NewSymbolLogger(sessionID string, baseDir string, level string) *SymbolLogger {
	zapLevel := zapcore.InfoLevel
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	}

	return &SymbolLogger{
		sessionID:  sessionID,
		baseDir:    baseDir,
		loggers:    make(map[string]*zap.Logger),
		level:      zapLevel,
		maxSize:    50, // 50MB per symbol
		maxBackups: 5,  // 保留5个备份
		maxAge:     30, // 30天
		compress:   true,
	}
}

// GetLogger 获取指定品种的日志记录器
func (s *SymbolLogger) GetLogger(symbol, interval string) *zap.Logger {
	key := fmt.Sprintf("%s_%s", symbol, interval)

	s.mu.RLock()
	logger, exists := s.loggers[key]
	s.mu.RUnlock()

	if exists {
		return logger
	}

	// 创建新的logger
	s.mu.Lock()
	defer s.mu.Unlock()

	// 双重检查
	if logger, exists := s.loggers[key]; exists {
		return logger
	}

	// 创建文件路径: logs/session_YYYYMMDD_HHMMSS/BTCUSDT_3m.log
	sessionDir := filepath.Join(s.baseDir, fmt.Sprintf("session_%s", s.sessionID))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		// 如果创建失败，返回全局logger
		return Logger
	}

	logFile := filepath.Join(sessionDir, fmt.Sprintf("%s.log", key))

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

	// 文件轮转
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    s.maxSize,
		MaxBackups: s.maxBackups,
		MaxAge:     s.maxAge,
		Compress:   s.compress,
	})

	// 控制台输出（可选）
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 创建核心
	core := zapcore.NewTee(
		// 文件输出：JSON格式
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			fileWriter,
			s.level,
		),
		// 控制台输出：简化格式
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderConfig),
			consoleWriter,
			zapcore.WarnLevel, // 控制台只显示警告以上
		),
	)

	// CallerSkip=0: 显示实际调用logger的位置（adapter.go）
	// 不跳过任何调用栈，显示真实的调用位置
	logger = zap.New(core, zap.AddCaller())
	s.loggers[key] = logger

	// 记录日志文件创建
	logger.Info("Log file created",
		zap.String("symbol", symbol),
		zap.String("interval", interval),
		zap.String("file", logFile),
	)

	return logger
}

// Close 关闭所有日志记录器
func (s *SymbolLogger) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, logger := range s.loggers {
		logger.Sync()
	}
}

// GetSessionDir 获取会话目录路径
func (s *SymbolLogger) GetSessionDir() string {
	return filepath.Join(s.baseDir, fmt.Sprintf("session_%s", s.sessionID))
}

// GetSessionID 获取会话ID
func (s *SymbolLogger) GetSessionID() string {
	return s.sessionID
}

// ========== 全局实例 ==========

var (
	globalSymbolLogger *SymbolLogger
	symbolLoggerMu     sync.Mutex
)

// InitSymbolLogger 初始化全局按品种分文件的日志记录器
func InitSymbolLogger(level string) *SymbolLogger {
	symbolLoggerMu.Lock()
	defer symbolLoggerMu.Unlock()

	if globalSymbolLogger != nil {
		return globalSymbolLogger
	}

	// 生成会话ID（时间戳）
	sessionID := time.Now().Format("20060102_150405")

	globalSymbolLogger = NewSymbolLogger(sessionID, "logs", level)
	return globalSymbolLogger
}

// GetSymbolLogger 获取全局按品种分文件的日志记录器
func GetSymbolLogger() *SymbolLogger {
	symbolLoggerMu.Lock()
	defer symbolLoggerMu.Unlock()

	if globalSymbolLogger == nil {
		globalSymbolLogger = InitSymbolLogger("info")
	}

	return globalSymbolLogger
}

// CloseSymbolLogger 关闭全局按品种分文件的日志记录器
func CloseSymbolLogger() {
	symbolLoggerMu.Lock()
	defer symbolLoggerMu.Unlock()

	if globalSymbolLogger != nil {
		globalSymbolLogger.Close()
		globalSymbolLogger = nil
	}
}
