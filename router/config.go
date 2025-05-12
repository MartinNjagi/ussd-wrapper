package router

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	// Server settings
	Port        string `mapstructure:"PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	// Database settings
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBUser     string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASSWORD"`
	DBName     string `mapstructure:"DB_NAME"`
	DBSSLMode  string `mapstructure:"DB_SSL_MODE"`

	// Slave database for read operations
	DBSlaveHost     string `mapstructure:"DB_SLAVE_HOST"`
	DBSlavePort     string `mapstructure:"DB_SLAVE_PORT"`
	DBSlaveUser     string `mapstructure:"DB_SLAVE_USER"`
	DBSlavePassword string `mapstructure:"DB_SLAVE_PASSWORD"`
	DBSlaveName     string `mapstructure:"DB_SLAVE_NAME"`
	DBSlaveSSLMode  string `mapstructure:"DB_SLAVE_SSL_MODE"`

	// Redis settings
	RedisHost     string `mapstructure:"REDIS_HOST"`
	RedisPort     string `mapstructure:"REDIS_PORT"`
	RedisPassword string `mapstructure:"REDIS_PASSWORD"`
	RedisDB       int    `mapstructure:"REDIS_DB"`

	// RabbitMQ settings
	RabbitMQURL string `mapstructure:"RABBITMQ_URL"`

	// JWT settings
	JWTSecret     string `mapstructure:"JWT_SECRET"`
	JWTExpiration int    `mapstructure:"JWT_EXPIRATION"` // In hours

	// Logging settings
	LogDir   string `mapstructure:"LOG_DIR"`
	LogLevel string `mapstructure:"LOG_LEVEL"`

	// Tracing settings
	TracingEnabled bool   `mapstructure:"TRACING_ENABLED"`
	JaegerEndpoint string `mapstructure:"JAEGER_ENDPOINT"`

	// USSD settings
	USSDShortcode  string `mapstructure:"USSD_SHORTCODE"`
	USSDSessionTTL int    `mapstructure:"USSD_SESSION_TTL"` // In minutes

	// SMS settings
	SMSEndpoint string `mapstructure:"SMS_ENDPOINT"`
	SMSAPIKey   string `mapstructure:"SMS_API_KEY"`

	// Transaction settings
	DefaultFee        float64 `mapstructure:"DEFAULT_FEE"`
	MinTransferAmount float64 `mapstructure:"MIN_TRANSFER_AMOUNT"`
	MaxTransferAmount float64 `mapstructure:"MAX_TRANSFER_AMOUNT"`

	// Admin settings
	AdminUsername string `mapstructure:"ADMIN_USERNAME"`
	AdminPassword string `mapstructure:"ADMIN_PASSWORD"`
}

// LoadConfig reads configuration from file or environment variables
func LoadConfig() (config Config, err error) {
	// Set defaults
	viper.SetDefault("PORT", "8080")
	viper.SetDefault("ENVIRONMENT", "development")
	viper.SetDefault("LOG_DIR", "./logs")
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("USSD_SESSION_TTL", 5) // 5 minutes
	viper.SetDefault("JWT_EXPIRATION", 24)  // 24 hours
	viper.SetDefault("DB_SSL_MODE", "disable")
	viper.SetDefault("DB_SLAVE_SSL_MODE", "disable")
	viper.SetDefault("TRACING_ENABLED", false)
	viper.SetDefault("DEFAULT_FEE", 0.00)
	viper.SetDefault("MIN_TRANSFER_AMOUNT", 10.00)
	viper.SetDefault("MAX_TRANSFER_AMOUNT", 100000.00)

	// Try to load .env file if it exists
	_ = godotenv.Load()

	// Set config type and name
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")

	// Add config locations to search
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AddConfigPath("/etc/ussd-wrapper")

	// Read environment variables with prefix
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file (non-fatal if not found)
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return config, fmt.Errorf("error reading config file: %w", err)
		}
		// Config file not found, will rely on environment variables
	}

	// Unmarshal config
	if err := viper.Unmarshal(&config); err != nil {
		return config, fmt.Errorf("unable to decode config: %w", err)
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(config.LogDir, 0755); err != nil {
		return config, fmt.Errorf("failed to create log directory: %w", err)
	}

	return config, nil
}

// String returns a redacted string representation of the config
func (c Config) String() string {
	// Create a copy with sensitive fields redacted
	copy := c
	copy.DBPassword = "[REDACTED]"
	copy.DBSlavePassword = "[REDACTED]"
	copy.RedisPassword = "[REDACTED]"
	copy.JWTSecret = "[REDACTED]"
	copy.SMSAPIKey = "[REDACTED]"
	copy.AdminPassword = "[REDACTED]"

	// Marshal to JSON for readable output
	bytes, _ := json.MarshalIndent(copy, "", "  ")
	return string(bytes)
}

// GetDBConnString returns the PostgreSQL connection string for the primary database
func (c Config) GetDBConnString() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBHost, c.DBPort, c.DBUser, c.DBPassword, c.DBName, c.DBSSLMode)
}

// GetDBSlaveConnString returns the PostgreSQL connection string for the slave/read database
func (c Config) GetDBSlaveConnString() string {
	if c.DBSlaveHost == "" {
		// If no slave configured, use the primary DB
		return c.GetDBConnString()
	}
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		c.DBSlaveHost, c.DBSlavePort, c.DBSlaveUser, c.DBSlavePassword, c.DBSlaveName, c.DBSlaveSSLMode)
}

// IsDevelopment returns true if the application is in development mode
func (c Config) IsDevelopment() bool {
	return strings.ToLower(c.Environment) == "development"
}

// Example config.yaml file (placed in ./config/config.yaml)
// -----------------------------------------------------------
/*
port: 8080
environment: development
log_dir: ./logs
log_level: debug

# Database settings
db_host: localhost
db_port: 5432
db_user: postgres
db_password: postgres
db_name: ussd_wrapper

# Redis settings
redis_host: localhost
redis_port: 6379
redis_password: ""
redis_db: 0

# RabbitMQ settings
rabbitmq_url: amqp://guest:guest@localhost:5672/

# JWT settings
jwt_secret: your-very-secret-key-here
jwt_expiration: 24

# USSD settings
ussd_shortcode: "*123#"
ussd_session_ttl: 5

# SMS settings
sms_endpoint: https://api.africastalking.com/version1/messaging
sms_api_key: your-sms-api-key-here
*/
