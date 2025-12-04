package v2

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// KlineStore 管理K线数据的数据库存储
type KlineStore struct {
	db       *sql.DB
	dbPath   string
	mu       sync.RWMutex
	prepared map[string]*sql.Stmt // 缓存预编译的SQL语句
}

// NewKlineStore 创建一个新的K线数据存储实例
// dbPath: 数据库文件路径，例如 "./data/klines.db"
func NewKlineStore(dbPath string) (*KlineStore, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	store := &KlineStore{
		db:       db,
		dbPath:   dbPath,
		prepared: make(map[string]*sql.Stmt),
	}

	// 初始化表
	if err := store.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("init tables: %w", err)
	}

	return store, nil
}

// initTables 创建K线数据表
func (ks *KlineStore) initTables() error {
	createTableSQL := `
	CREATE TABLE IF NOT EXISTS klines (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		event_type TEXT NOT NULL,
		event_time INTEGER NOT NULL,
		symbol TEXT NOT NULL,
		start_time INTEGER NOT NULL,
		close_time INTEGER NOT NULL,
		interval TEXT NOT NULL,
		open_price REAL NOT NULL,
		close_price REAL NOT NULL,
		high_price REAL NOT NULL,
		low_price REAL NOT NULL,
		base_volume REAL NOT NULL,
		quote_volume REAL NOT NULL,
		is_closed INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(symbol, interval, close_time)
	);
	`

	// 创建索引以加快查询速度
	createIndexSQL := []string{
		`CREATE INDEX IF NOT EXISTS idx_klines_symbol_interval 
		 ON klines(symbol, interval)`,
		`CREATE INDEX IF NOT EXISTS idx_klines_close_time 
		 ON klines(close_time)`,
		`CREATE INDEX IF NOT EXISTS idx_klines_symbol_interval_close_time 
		 ON klines(symbol, interval, close_time)`,
	}

	_, err := ks.db.Exec(createTableSQL)
	if err != nil {
		return fmt.Errorf("create klines table: %w", err)
	}

	for _, indexSQL := range createIndexSQL {
		_, err := ks.db.Exec(indexSQL)
		if err != nil {
			return fmt.Errorf("create index: %w", err)
		}
	}

	return nil
}

// SaveKline 保存一条K线数据到数据库
func (ks *KlineStore) SaveKline(kline *KlineData) error {
	if kline == nil {
		return fmt.Errorf("kline data is nil")
	}

	insertSQL := `
	INSERT OR REPLACE INTO klines (
		event_type, event_time, symbol, start_time, close_time, interval,
		open_price, close_price, high_price, low_price, base_volume, quote_volume, is_closed
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	_, err := ks.db.Exec(
		insertSQL,
		kline.EventType,
		kline.EventTime,
		kline.Symbol,
		kline.StartTime,
		kline.CloseTime,
		kline.Interval,
		kline.OpenPrice,
		kline.ClosePrice,
		kline.HighPrice,
		kline.LowPrice,
		kline.BaseVolume,
		kline.QuoteVolume,
		boolToInt(kline.IsClosed),
	)

	if err != nil {
		return fmt.Errorf("insert kline: %w", err)
	}

	return nil
}

// SaveKlines 批量保存K线数据（事务）
func (ks *KlineStore) SaveKlines(klines []*KlineData) error {
	if len(klines) == 0 {
		return nil
	}

	tx, err := ks.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	insertSQL := `
	INSERT OR REPLACE INTO klines (
		event_type, event_time, symbol, start_time, close_time, interval,
		open_price, close_price, high_price, low_price, base_volume, quote_volume, is_closed
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, kline := range klines {
		if kline == nil {
			continue
		}
		_, err := stmt.Exec(
			kline.EventType,
			kline.EventTime,
			kline.Symbol,
			kline.StartTime,
			kline.CloseTime,
			kline.Interval,
			kline.OpenPrice,
			kline.ClosePrice,
			kline.HighPrice,
			kline.LowPrice,
			kline.BaseVolume,
			kline.QuoteVolume,
			boolToInt(kline.IsClosed),
		)
		if err != nil {
			return fmt.Errorf("insert kline: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// GetKlines 查询K线数据
// symbol: 交易对，例如 "BTCUSDT"
// interval: 时间间隔，例如 "1m", "5m"
// limit: 限制返回条数，0表示不限制
func (ks *KlineStore) GetKlines(symbol, interval string, limit int) ([]*KlineData, error) {
	query := `
	SELECT event_type, event_time, symbol, start_time, close_time, interval,
	       open_price, close_price, high_price, low_price, base_volume, quote_volume, is_closed
	FROM klines
	WHERE symbol = ? AND interval = ?
	ORDER BY close_time DESC
	`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := ks.db.Query(query, symbol, interval)
	if err != nil {
		return nil, fmt.Errorf("query klines: %w", err)
	}
	defer rows.Close()

	var klines []*KlineData
	for rows.Next() {
		var isClosed int
		kline := &KlineData{}
		err := rows.Scan(
			&kline.EventType,
			&kline.EventTime,
			&kline.Symbol,
			&kline.StartTime,
			&kline.CloseTime,
			&kline.Interval,
			&kline.OpenPrice,
			&kline.ClosePrice,
			&kline.HighPrice,
			&kline.LowPrice,
			&kline.BaseVolume,
			&kline.QuoteVolume,
			&isClosed,
		)
		if err != nil {
			return nil, fmt.Errorf("scan kline: %w", err)
		}
		kline.IsClosed = isClosed == 1
		klines = append(klines, kline)
	}

	return klines, rows.Err()
}

// GetKlineCount 获取K线数据总数
func (ks *KlineStore) GetKlineCount(symbol, interval string) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM klines WHERE symbol = ? AND interval = ?"
	err := ks.db.QueryRow(query, symbol, interval).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count klines: %w", err)
	}
	return count, nil
}

// Close 关闭数据库连接
func (ks *KlineStore) Close() error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	// 关闭所有预编译的语句
	for _, stmt := range ks.prepared {
		stmt.Close()
	}

	if ks.db != nil {
		return ks.db.Close()
	}
	return nil
}

// 辅助函数
func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
