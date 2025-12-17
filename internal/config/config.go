package config

import (
	"fmt"
	"time"

	"goQuant/internal/core"

	"github.com/spf13/viper"
)

// Config 全局配置
type Config struct {
	App           AppConfig           `mapstructure:"app"`
	Data          DataConfig          `mapstructure:"data"`
	Strategy      StrategyConfig      `mapstructure:"strategy"`
	Position      PositionConfig      `mapstructure:"position"`
	Execution     ExecutionConfig     `mapstructure:"execution"`
	Observability ObservabilityConfig `mapstructure:"observability"`
}

// AppConfig 应用配置
type AppConfig struct {
	Name        string `mapstructure:"name"`
	Mode        string `mapstructure:"mode"`        // "backtest", "paper", "live"
	Environment string `mapstructure:"environment"` // "development", "production"
}

// DataConfig 数据模块配置
type DataConfig struct {
	Provider      string          `mapstructure:"provider"` // "binance"
	ProxyURL      string          `mapstructure:"proxy_url"`
	DatabaseDir   string          `mapstructure:"database_dir"`
	Subscriptions []Subscription  `mapstructure:"subscriptions"`
	Reconnect     ReconnectConfig `mapstructure:"reconnect"`
}

// Subscription 订阅配置
type Subscription struct {
	Symbol          string `mapstructure:"symbol"`
	Interval        string `mapstructure:"interval"`
	EnableOrderBook bool   `mapstructure:"enable_orderbook"`
	OrderBookLevels int    `mapstructure:"orderbook_levels"`
}

// ReconnectConfig 重连配置
type ReconnectConfig struct {
	MaxRetries       int           `mapstructure:"max_retries"`
	InitialDelay     time.Duration `mapstructure:"initial_delay"`
	MaxDelay         time.Duration `mapstructure:"max_delay"`
	HeartbeatTimeout time.Duration `mapstructure:"heartbeat_timeout"`
}

// StrategyConfig 策略配置
type StrategyConfig struct {
	Name           string                 `mapstructure:"name"`
	WarmupPeriods  int                    `mapstructure:"warmup_periods"`
	Parameters     map[string]interface{} `mapstructure:"parameters"`
	EnableBackfill bool                   `mapstructure:"enable_backfill"` // 启动时是否回填历史数据
}

// PositionConfig 仓位管理配置
type PositionConfig struct {
	DefaultLeverage   int                  `mapstructure:"default_leverage"`
	DefaultMarginMode core.MarginMode      `mapstructure:"default_margin_mode"`
	MaxPositionSize   float64              `mapstructure:"max_position_size"` // 最大仓位（占总资金比例）
	RiskLimits        RiskLimitsConfig     `mapstructure:"risk_limits"`
	PositionSizing    PositionSizingConfig `mapstructure:"position_sizing"`
}

// RiskLimitsConfig 风险限制配置
type RiskLimitsConfig struct {
	MaxDrawdown       float64 `mapstructure:"max_drawdown"`        // 最大回撤（百分比）
	MaxDailyLoss      float64 `mapstructure:"max_daily_loss"`      // 单日最大亏损
	MaxOpenPositions  int     `mapstructure:"max_open_positions"`  // 最大持仓数
	StopLossPercent   float64 `mapstructure:"stop_loss_percent"`   // 止损百分比
	TakeProfitPercent float64 `mapstructure:"take_profit_percent"` // 止盈百分比
}

// PositionSizingConfig 仓位计算配置
type PositionSizingConfig struct {
	Method       string  `mapstructure:"method"`        // "fixed", "risk_based", "kelly"
	FixedPercent float64 `mapstructure:"fixed_percent"` // 固定百分比（如20%）
	OpenPercent  float64 `mapstructure:"open_percent"`  // 开仓百分比
	AddPercent   float64 `mapstructure:"add_percent"`   // 加仓百分比
	RiskPercent  float64 `mapstructure:"risk_percent"`  // 风险百分比
}

// ExecutionConfig 执行模块配置
type ExecutionConfig struct {
	Mode     string         `mapstructure:"mode"`     // "backtest", "paper", "live"
	Exchange string         `mapstructure:"exchange"` // "binance"
	Binance  BinanceConfig  `mapstructure:"binance"`
	Backtest BacktestConfig `mapstructure:"backtest"`
	Fees     FeesConfig     `mapstructure:"fees"`
	Slippage SlippageConfig `mapstructure:"slippage"`
}

// BinanceConfig 币安API配置
type BinanceConfig struct {
	APIKey     string `mapstructure:"api_key"`
	SecretKey  string `mapstructure:"secret_key"`
	BaseURL    string `mapstructure:"base_url"`
	WSBaseURL  string `mapstructure:"ws_base_url"`
	Testnet    bool   `mapstructure:"testnet"`
	UseWSOrder bool   `mapstructure:"use_ws_order"` // 是否使用 WebSocket 下单（默认 false）
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	DataSource     string  `mapstructure:"data_source"` // "database", "csv"
	DatabasePath   string  `mapstructure:"database_path"`
	StartDate      string  `mapstructure:"start_date"` // 格式: "2024-01-01T00:00:00Z"
	EndDate        string  `mapstructure:"end_date"`   // 格式: "2024-12-31T23:59:59Z"
	InitialBalance float64 `mapstructure:"initial_balance"`
}

// GetStartTime 获取开始时间
func (b *BacktestConfig) GetStartTime() (time.Time, error) {
	if b.StartDate == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, b.StartDate)
}

// GetEndTime 获取结束时间
func (b *BacktestConfig) GetEndTime() (time.Time, error) {
	if b.EndDate == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, b.EndDate)
}

// FeesConfig 手续费配置
type FeesConfig struct {
	MakerFee float64 `mapstructure:"maker_fee"` // Maker手续费率
	TakerFee float64 `mapstructure:"taker_fee"` // Taker手续费率
}

// SlippageConfig 滑点配置
type SlippageConfig struct {
	Enabled   bool `mapstructure:"enabled"`
	FixedBps  int  `mapstructure:"fixed_bps"`  // 固定滑点（基点）
	RandomBps int  `mapstructure:"random_bps"` // 随机滑点范围
}

// ObservabilityConfig 可观测性配置
type ObservabilityConfig struct {
	Logging LoggingConfig `mapstructure:"logging"`
	Metrics MetricsConfig `mapstructure:"metrics"`
	Alerts  AlertsConfig  `mapstructure:"alerts"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level      string `mapstructure:"level"`       // "debug", "info", "warn", "error"
	Format     string `mapstructure:"format"`      // "json", "text"
	OutputPath string `mapstructure:"output_path"` // 文件路径或"stdout"
	MaxSize    int    `mapstructure:"max_size"`    // MB
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"` // days
}

// MetricsConfig 指标配置
type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Port    int    `mapstructure:"port"` // Prometheus端口
	Path    string `mapstructure:"path"` // Metrics路径
}

// AlertsConfig 告警配置
type AlertsConfig struct {
	Enabled   bool        `mapstructure:"enabled"`
	Channels  []string    `mapstructure:"channels"` // "console", "file", "sns"
	SNSConfig SNSConfig   `mapstructure:"sns"`
	Rules     []AlertRule `mapstructure:"rules"`
}

// SNSConfig SNS通知配置（预留）
type SNSConfig struct {
	Region   string `mapstructure:"region"`
	TopicARN string `mapstructure:"topic_arn"`
	Enabled  bool   `mapstructure:"enabled"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name      string `mapstructure:"name"`
	Condition string `mapstructure:"condition"` // 如 "drawdown > 0.05"
	Severity  string `mapstructure:"severity"`  // "info", "warning", "critical"
}

// Load 加载配置文件
func Load(configPath string) (*Config, error) {
	v := viper.New()

	// 设置配置文件路径
	if configPath != "" {
		v.SetConfigFile(configPath)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.AddConfigPath("$HOME/.goquant")
	}

	// 设置默认值
	setDefaults(v)

	// 读取环境变量
	v.AutomaticEnv()
	v.SetEnvPrefix("GOQUANT")

	// 读取配置文件
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// 解析配置
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	// 验证配置
	if err := validate(&config); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &config, nil
}

// setDefaults 设置默认值
func setDefaults(v *viper.Viper) {
	// App
	v.SetDefault("app.name", "goQuant")
	v.SetDefault("app.mode", "backtest")
	v.SetDefault("app.environment", "development")

	// Data
	v.SetDefault("data.provider", "binance")
	v.SetDefault("data.database_dir", "./data/wsdata")
	v.SetDefault("data.reconnect.max_retries", 10)
	v.SetDefault("data.reconnect.initial_delay", "1s")
	v.SetDefault("data.reconnect.max_delay", "5m")
	v.SetDefault("data.reconnect.heartbeat_timeout", "45s")

	// Position
	v.SetDefault("position.default_leverage", 5)
	v.SetDefault("position.default_margin_mode", "ISOLATED")
	v.SetDefault("position.max_position_size", 0.5)
	v.SetDefault("position.risk_limits.max_drawdown", 0.2)
	v.SetDefault("position.risk_limits.max_daily_loss", 0.05)
	v.SetDefault("position.risk_limits.max_open_positions", 3)
	v.SetDefault("position.position_sizing.method", "fixed")
	v.SetDefault("position.position_sizing.open_percent", 0.2)
	v.SetDefault("position.position_sizing.add_percent", 0.4)

	// Execution
	v.SetDefault("execution.mode", "backtest")
	v.SetDefault("execution.exchange", "binance")
	v.SetDefault("execution.fees.maker_fee", 0.0002)
	v.SetDefault("execution.fees.taker_fee", 0.0004)
	v.SetDefault("execution.slippage.enabled", true)
	v.SetDefault("execution.slippage.fixed_bps", 5)
	v.SetDefault("execution.backtest.initial_balance", 10000.0)

	// Observability
	v.SetDefault("observability.logging.level", "info")
	v.SetDefault("observability.logging.format", "json")
	v.SetDefault("observability.logging.output_path", "stdout")
	v.SetDefault("observability.metrics.enabled", true)
	v.SetDefault("observability.metrics.port", 9090)
	v.SetDefault("observability.metrics.path", "/metrics")
	v.SetDefault("observability.alerts.enabled", true)
	v.SetDefault("observability.alerts.channels", []string{"console"})
}

// validate 验证配置
func validate(cfg *Config) error {
	// 验证应用模式
	validModes := map[string]bool{"backtest": true, "paper": true, "live": true}
	if !validModes[cfg.App.Mode] {
		return fmt.Errorf("invalid app mode: %s", cfg.App.Mode)
	}

	// 验证杠杆范围
	if cfg.Position.DefaultLeverage < 1 || cfg.Position.DefaultLeverage > 125 {
		return fmt.Errorf("invalid leverage: %d (must be 1-125)", cfg.Position.DefaultLeverage)
	}

	// 验证风险限制
	if cfg.Position.RiskLimits.MaxDrawdown <= 0 || cfg.Position.RiskLimits.MaxDrawdown > 1 {
		return fmt.Errorf("invalid max_drawdown: %.2f (must be 0-1)", cfg.Position.RiskLimits.MaxDrawdown)
	}

	// 实盘模式需要API密钥
	if cfg.App.Mode == "live" {
		if cfg.Execution.Binance.APIKey == "" || cfg.Execution.Binance.SecretKey == "" {
			return fmt.Errorf("live mode requires API key and secret")
		}
	}

	return nil
}
