package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server     ServerConfig
	LogLevel   string           `mapstructure:"log_level"`
	RabbitMQ   RabbitMQConfig   `mapstructure:"rabbitmq"`
	MongoDB    MongoDBConfig    `mapstructure:"mongodb"`
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	Security   SecurityConfig   `mapstructure:"security"`
}

type SecurityConfig struct {
	APIKeyHeader string            `mapstructure:"apiKeyHeader"`
	APIKeys      map[string]string `mapstructure:"apiKeys"`
}

type MonitoringConfig struct {
	PrometheusPort int    `mapstructure:"prometheusPort"`
	MetricsPath    string `mapstructure:"metricsPath"`
}

type MongoDBConfig struct {
	URI        string `mapstructure:"uri"`
	Database   string `mapstructure:"database"`
	Collection string `mapstructure:"collection"`
}

type RabbitMQConfig struct {
	URL       string `mapstructure:"url"`
	Exchange  string `mapstructure:"exchange"`
	QueueName string `mapstructure:"queueName"`
}

type ServerConfig struct {
	Port int
	Host string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("server.host", "0.0.0.0")
	viper.SetDefault("log_level", "info")
	viper.SetDefault("monitoring.prometheusPort", 9090)
	viper.SetDefault("monitoring.metricsPath", "/metrics")

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// Override with environment variables
	if port := os.Getenv("APP_PORT"); port != "" {
		if p, err := strconv.Atoi(port); err == nil {
			cfg.Server.Port = p
		}
	}

	if promPort := os.Getenv("PROMETHEUS_PORT"); promPort != "" {
		if p, err := strconv.Atoi(promPort); err == nil {
			cfg.Monitoring.PrometheusPort = p
		}
	}

	if uri := os.Getenv("MONGODB_URI"); uri != "" {
		cfg.MongoDB.URI = uri
	}
	if db := os.Getenv("MONGODB_DATABASE"); db != "" {
		cfg.MongoDB.Database = db
	}
	if col := os.Getenv("MONGODB_COLLECTION"); col != "" {
		cfg.MongoDB.Collection = col
	}

	// Support both CLOUDAMQP_URL and RABBITMQ_URI for backwards compatibility
	if cloudamqpURL := os.Getenv("CLOUDAMQP_URL"); cloudamqpURL != "" {
		cfg.RabbitMQ.URL = cloudamqpURL
	} else if rabbitURL := os.Getenv("RABBITMQ_URI"); rabbitURL != "" {
		cfg.RabbitMQ.URL = rabbitURL
	}

	if exchange := os.Getenv("RABBITMQ_EXCHANGE"); exchange != "" {
		cfg.RabbitMQ.Exchange = exchange
	}
	if queue := os.Getenv("RABBITMQ_QUEUE"); queue != "" {
		cfg.RabbitMQ.QueueName = queue
	}

	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.LogLevel = level
	}

	if header := os.Getenv("API_KEY_HEADER"); header != "" {
		cfg.Security.APIKeyHeader = header
	}

	// Load API keys from environment
	cfg.Security.APIKeys = loadAPIKeysFromEnv()

	return &cfg, nil
}

func loadAPIKeysFromEnv() map[string]string {
	apiKeys := make(map[string]string)

	// Load specific client API keys
	if key := os.Getenv("MAILERCLOUD_API_KEY"); key != "" {
		apiKeys["mailercloud"] = key
	}

	// Load other client API keys from environment variables with pattern CLIENT_*_API_KEY
	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			continue
		}

		envName := parts[0]
		envValue := parts[1]

		if strings.HasSuffix(envName, "_API_KEY") && envName != "MAILERCLOUD_API_KEY" {
			// Convert CLIENT_NAME_API_KEY to client_name
			clientName := strings.ToLower(strings.TrimSuffix(envName, "_API_KEY"))
			apiKeys[clientName] = envValue
		}
	}

	return apiKeys
}
