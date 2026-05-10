package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"gopkg.in/yaml.v3"
)

type duration time.Duration

func (d *duration) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return fmt.Errorf("duration must be a scalar string, got kind %d", node.Kind)
	}
	parsed, err := time.ParseDuration(node.Value)
	if err != nil {
		return fmt.Errorf("parse duration %q: %w", node.Value, err)
	}
	*d = duration(parsed)
	return nil
}

type Settings struct {
	Kafka    KafkaConfig    `yaml:"kafka"`
	Redis    RedisConfig    `yaml:"redis"`
	Postgres PostgresConfig `yaml:"postgres"`
	HTTP     HTTPConfig     `yaml:"http"`
	WS       WSConfig       `yaml:"ws"`
	Runtime  RuntimeConfig  `yaml:"runtime"`
}

type KafkaConfig struct {
	BootstrapServers          []string      `yaml:"bootstrap_servers"`
	GroupID                   string        `yaml:"group_id"`
	TopicMarketDataNormalized string        `yaml:"topic_market_data_normalized"`
	TopicTradingSignals       string        `yaml:"topic_trading_signals"`
	TopicSystemEvents         string        `yaml:"topic_system_events"`
	ReconnectWait             time.Duration `yaml:"reconnect_wait"`
	ReconnectMaxWait          time.Duration `yaml:"reconnect_max_wait"`
	MinBytes                  int           `yaml:"min_bytes"`
	MaxBytes                  int           `yaml:"max_bytes"`
}

func (k *KafkaConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		BootstrapServers          []string `yaml:"bootstrap_servers"`
		GroupID                   string   `yaml:"group_id"`
		TopicMarketDataNormalized string   `yaml:"topic_market_data_normalized"`
		TopicTradingSignals       string   `yaml:"topic_trading_signals"`
		TopicSystemEvents         string   `yaml:"topic_system_events"`
		ReconnectWait             duration `yaml:"reconnect_wait"`
		ReconnectMaxWait          duration `yaml:"reconnect_max_wait"`
		MinBytes                  int      `yaml:"min_bytes"`
		MaxBytes                  int      `yaml:"max_bytes"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*k = KafkaConfig{
		BootstrapServers:          r.BootstrapServers,
		GroupID:                   r.GroupID,
		TopicMarketDataNormalized: r.TopicMarketDataNormalized,
		TopicTradingSignals:       r.TopicTradingSignals,
		TopicSystemEvents:         r.TopicSystemEvents,
		ReconnectWait:             time.Duration(r.ReconnectWait),
		ReconnectMaxWait:          time.Duration(r.ReconnectMaxWait),
		MinBytes:                  r.MinBytes,
		MaxBytes:                  r.MaxBytes,
	}
	return nil
}

type RedisConfig struct {
	Host         string        `yaml:"host"`
	Port         int           `yaml:"port"`
	DB           int           `yaml:"db"`
	Password     string        `yaml:"password"`
	PoolSize     int           `yaml:"pool_size"`
	MinIdleConns int           `yaml:"min_idle_conns"`
	DialTimeout  time.Duration `yaml:"dial_timeout"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`

	QuotePrefix     string `yaml:"quote_prefix"`
	IndicatorPrefix string `yaml:"indicator_prefix"`
	TechPrefix      string `yaml:"tech_prefix"`
	CapitalPrefix   string `yaml:"capital_prefix"`
	FenbiPrefix     string `yaml:"fenbi_prefix"`
	FinancePrefix   string `yaml:"finance_prefix"`
	HistDailyPrefix string `yaml:"hist_daily_prefix"`
	StockListKey    string `yaml:"stock_list_key"`
}

func (rc *RedisConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		Host            string   `yaml:"host"`
		Port            int      `yaml:"port"`
		DB              int      `yaml:"db"`
		Password        string   `yaml:"password"`
		PoolSize        int      `yaml:"pool_size"`
		MinIdleConns    int      `yaml:"min_idle_conns"`
		DialTimeout     duration `yaml:"dial_timeout"`
		ReadTimeout     duration `yaml:"read_timeout"`
		WriteTimeout    duration `yaml:"write_timeout"`
		QuotePrefix     string   `yaml:"quote_prefix"`
		IndicatorPrefix string   `yaml:"indicator_prefix"`
		TechPrefix      string   `yaml:"tech_prefix"`
		CapitalPrefix   string   `yaml:"capital_prefix"`
		FenbiPrefix     string   `yaml:"fenbi_prefix"`
		FinancePrefix   string   `yaml:"finance_prefix"`
		HistDailyPrefix string   `yaml:"hist_daily_prefix"`
		StockListKey    string   `yaml:"stock_list_key"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*rc = RedisConfig{
		Host:            r.Host,
		Port:            r.Port,
		DB:              r.DB,
		Password:        r.Password,
		PoolSize:        r.PoolSize,
		MinIdleConns:    r.MinIdleConns,
		DialTimeout:     time.Duration(r.DialTimeout),
		ReadTimeout:     time.Duration(r.ReadTimeout),
		WriteTimeout:    time.Duration(r.WriteTimeout),
		QuotePrefix:     r.QuotePrefix,
		IndicatorPrefix: r.IndicatorPrefix,
		TechPrefix:      r.TechPrefix,
		CapitalPrefix:   r.CapitalPrefix,
		FenbiPrefix:     r.FenbiPrefix,
		FinancePrefix:   r.FinancePrefix,
		HistDailyPrefix: r.HistDailyPrefix,
		StockListKey:    r.StockListKey,
	}
	return nil
}

func (r RedisConfig) Quote(symbol string) string     { return r.QuotePrefix + symbol }
func (r RedisConfig) Indicator(symbol string) string { return r.IndicatorPrefix + symbol }
func (r RedisConfig) Tech(symbol string) string      { return r.TechPrefix + symbol }
func (r RedisConfig) Capital(symbol string) string   { return r.CapitalPrefix + symbol }
func (r RedisConfig) Fenbi(symbol string) string     { return r.FenbiPrefix + symbol }
func (r RedisConfig) Finance(symbol string) string   { return r.FinancePrefix + symbol }
func (r RedisConfig) HistDaily(symbol string) string { return r.HistDailyPrefix + symbol }

type PostgresConfig struct {
	Host            string        `yaml:"host"`
	Port            int           `yaml:"port"`
	User            string        `yaml:"user"`
	Password        string        `yaml:"password"`
	Database        string        `yaml:"database"`
	SSLMode         string        `yaml:"sslmode"`
	MaxOpenConns    int           `yaml:"max_open_conns"`
	MaxIdleConns    int           `yaml:"max_idle_conns"`
	ConnMaxLifetime time.Duration `yaml:"conn_max_lifetime"`
}

func (p *PostgresConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		Host            string   `yaml:"host"`
		Port            int      `yaml:"port"`
		User            string   `yaml:"user"`
		Password        string   `yaml:"password"`
		Database        string   `yaml:"database"`
		SSLMode         string   `yaml:"sslmode"`
		MaxOpenConns    int      `yaml:"max_open_conns"`
		MaxIdleConns    int      `yaml:"max_idle_conns"`
		ConnMaxLifetime duration `yaml:"conn_max_lifetime"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*p = PostgresConfig{
		Host:            r.Host,
		Port:            r.Port,
		User:            r.User,
		Password:        r.Password,
		Database:        r.Database,
		SSLMode:         r.SSLMode,
		MaxOpenConns:    r.MaxOpenConns,
		MaxIdleConns:    r.MaxIdleConns,
		ConnMaxLifetime: time.Duration(r.ConnMaxLifetime),
	}
	return nil
}

func (p PostgresConfig) DSN() string {
	sslmode := p.SSLMode
	if sslmode == "" {
		sslmode = "disable"
	}
	if p.Password == "" {
		return fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
			p.Host, p.Port, p.User, p.Database, sslmode)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		p.Host, p.Port, p.User, p.Password, p.Database, sslmode)
}

type HTTPConfig struct {
	ListenAddr      string        `yaml:"listen_addr"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

func (h *HTTPConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		ListenAddr      string   `yaml:"listen_addr"`
		ReadTimeout     duration `yaml:"read_timeout"`
		WriteTimeout    duration `yaml:"write_timeout"`
		ShutdownTimeout duration `yaml:"shutdown_timeout"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*h = HTTPConfig{
		ListenAddr:      r.ListenAddr,
		ReadTimeout:     time.Duration(r.ReadTimeout),
		WriteTimeout:    time.Duration(r.WriteTimeout),
		ShutdownTimeout: time.Duration(r.ShutdownTimeout),
	}
	return nil
}

type WSConfig struct {
	ReadTimeout        time.Duration `yaml:"read_timeout"`
	WriteTimeout       time.Duration `yaml:"write_timeout"`
	PingInterval       time.Duration `yaml:"ping_interval"`
	QuoteFlushInterval time.Duration `yaml:"quote_flush_interval"`
	ClientSendBuffer   int           `yaml:"client_send_buffer"`
	MaxConnections     int           `yaml:"max_connections"`
	ReadBufferSize     int           `yaml:"read_buffer_size"`
	WriteBufferSize    int           `yaml:"write_buffer_size"`
}

func (w *WSConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		ReadTimeout        duration `yaml:"read_timeout"`
		WriteTimeout       duration `yaml:"write_timeout"`
		PingInterval       duration `yaml:"ping_interval"`
		QuoteFlushInterval duration `yaml:"quote_flush_interval"`
		ClientSendBuffer   int      `yaml:"client_send_buffer"`
		MaxConnections     int      `yaml:"max_connections"`
		ReadBufferSize     int      `yaml:"read_buffer_size"`
		WriteBufferSize    int      `yaml:"write_buffer_size"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*w = WSConfig{
		ReadTimeout:        time.Duration(r.ReadTimeout),
		WriteTimeout:       time.Duration(r.WriteTimeout),
		PingInterval:       time.Duration(r.PingInterval),
		QuoteFlushInterval: time.Duration(r.QuoteFlushInterval),
		ClientSendBuffer:   r.ClientSendBuffer,
		MaxConnections:     r.MaxConnections,
		ReadBufferSize:     r.ReadBufferSize,
		WriteBufferSize:    r.WriteBufferSize,
	}
	return nil
}

type RuntimeConfig struct {
	LogLevel        string        `yaml:"log_level"`
	MetricsEnabled  bool          `yaml:"metrics_enabled"`
	MetricsInterval time.Duration `yaml:"metrics_interval"`
}

func (rc *RuntimeConfig) UnmarshalYAML(node *yaml.Node) error {
	type raw struct {
		LogLevel        string   `yaml:"log_level"`
		MetricsEnabled  bool     `yaml:"metrics_enabled"`
		MetricsInterval duration `yaml:"metrics_interval"`
	}
	var r raw
	if err := node.Decode(&r); err != nil {
		return err
	}
	*rc = RuntimeConfig{
		LogLevel:        r.LogLevel,
		MetricsEnabled:  r.MetricsEnabled,
		MetricsInterval: time.Duration(r.MetricsInterval),
	}
	return nil
}

func Load() (*Settings, error) {
	path := resolveConfigPath()
	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config yaml %s: %w", path, err)
	}
	var s Settings
	if err := yaml.Unmarshal(buf, &s); err != nil {
		return nil, fmt.Errorf("parse config yaml %s: %w", path, err)
	}
	if err := validate(&s); err != nil {
		return nil, fmt.Errorf("validate config in %s: %w", path, err)
	}
	return &s, nil
}

func resolveConfigPath() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "config.yaml"
	}
	return filepath.Join(filepath.Dir(file), "config.yaml")
}

func validate(s *Settings) error {
	if len(s.Kafka.BootstrapServers) == 0 {
		return fmt.Errorf("kafka.bootstrap_servers must not be empty")
	}
	if s.Postgres.Database == "" {
		return fmt.Errorf("postgres.database must not be empty")
	}
	if s.HTTP.ListenAddr == "" {
		return fmt.Errorf("http.listen_addr must not be empty")
	}
	return nil
}
