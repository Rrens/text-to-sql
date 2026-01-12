package config

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all application configuration
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Vault    VaultConfig    `mapstructure:"vault"`
	Auth     AuthConfig     `mapstructure:"auth"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Security SecurityConfig `mapstructure:"security"`
	Logging  LoggingConfig  `mapstructure:"logging"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
}

type ServerConfig struct {
	Host         string        `mapstructure:"host"`
	Port         int           `mapstructure:"port"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

type DatabaseConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Database string `mapstructure:"database"`
	SSLMode  string `mapstructure:"ssl_mode"`
	MaxConns int32  `mapstructure:"max_conns"`
	MinConns int32  `mapstructure:"min_conns"`
}

func (c DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Database, c.SSLMode,
	)
}

type RedisConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

func (c RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

type VaultConfig struct {
	Address string `mapstructure:"address"`
	Token   string `mapstructure:"token"`
}

type AuthConfig struct {
	JWTSecret       string        `mapstructure:"jwt_secret"`
	AccessTokenTTL  time.Duration `mapstructure:"access_token_ttl"`
	RefreshTokenTTL time.Duration `mapstructure:"refresh_token_ttl"`
}

type LLMConfig struct {
	DefaultProvider string          `mapstructure:"default_provider"`
	OpenAI          OpenAIConfig    `mapstructure:"openai"`
	Anthropic       AnthropicConfig `mapstructure:"anthropic"`
	Ollama          OllamaConfig    `mapstructure:"ollama"`
	DeepSeek        DeepSeekConfig  `mapstructure:"deepseek"`
}

type OpenAIConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type AnthropicConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type OllamaConfig struct {
	Host         string `mapstructure:"host"`
	DefaultModel string `mapstructure:"default_model"`
}

type DeepSeekConfig struct {
	APIKey string `mapstructure:"api_key"`
	Model  string `mapstructure:"model"`
}

type SecurityConfig struct {
	ReadOnlyDefault bool            `mapstructure:"read_only_default"`
	MaxRows         int             `mapstructure:"max_rows"`
	QueryTimeout    time.Duration   `mapstructure:"query_timeout"`
	RateLimit       RateLimitConfig `mapstructure:"rate_limit"`
}

type RateLimitConfig struct {
	RequestsPerMinute int `mapstructure:"requests_per_minute"`
	Burst             int `mapstructure:"burst"`
}

type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

type MetricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Path    string `mapstructure:"path"`
}

// Load reads configuration from file and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set config file path
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "./configs/config.yaml"
	}

	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}
		// Config file not found, use defaults and env vars
	}

	// Override with environment variables
	v.AutomaticEnv()
	bindEnvVars(v)

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "30s")
	v.SetDefault("server.write_timeout", "30s")

	// Database
	v.SetDefault("database.host", "localhost")
	v.SetDefault("database.port", 5432)
	v.SetDefault("database.user", "texttosql")
	v.SetDefault("database.database", "texttosql")
	v.SetDefault("database.ssl_mode", "disable")
	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 5)

	// Redis
	v.SetDefault("redis.host", "localhost")
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.db", 0)

	// Auth
	v.SetDefault("auth.access_token_ttl", "15m")
	v.SetDefault("auth.refresh_token_ttl", "168h") // 7 days

	// LLM
	v.SetDefault("llm.default_provider", "ollama")
	v.SetDefault("llm.ollama.host", "http://localhost:11434")
	v.SetDefault("llm.ollama.default_model", "llama3")

	// Security
	v.SetDefault("security.read_only_default", true)
	v.SetDefault("security.max_rows", 1000)
	v.SetDefault("security.query_timeout", "30s")
	v.SetDefault("security.rate_limit.requests_per_minute", 60)
	v.SetDefault("security.rate_limit.burst", 10)

	// Logging
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")

	// Metrics
	v.SetDefault("metrics.enabled", true)
	v.SetDefault("metrics.path", "/metrics")
}

func bindEnvVars(v *viper.Viper) {
	// Database
	v.BindEnv("database.password", "POSTGRES_PASSWORD")

	// Redis
	v.BindEnv("redis.password", "REDIS_PASSWORD")

	// Vault
	v.BindEnv("vault.address", "VAULT_ADDR")
	v.BindEnv("vault.token", "VAULT_TOKEN")

	// Auth
	v.BindEnv("auth.jwt_secret", "JWT_SECRET")

	// LLM API Keys
	v.BindEnv("llm.openai.api_key", "OPENAI_API_KEY")
	v.BindEnv("llm.anthropic.api_key", "ANTHROPIC_API_KEY")
	v.BindEnv("llm.deepseek.api_key", "DEEPSEEK_API_KEY")
	v.BindEnv("llm.ollama.host", "OLLAMA_HOST")
}
