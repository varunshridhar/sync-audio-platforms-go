package config

import (
	"log"

	"github.com/spf13/viper"
)

// Global configuration variables.
var (
	// App holds the application-wide settings.
	App AppConfig
	// MySQL holds the MySQL connection configuration.
	MySQL MySQLConfig
	// Redis holds the Redis connection configuration.
	Redis RedisConfig
)

// LoadConfig initializes the configuration by reading from config.yml (if it exists)
// and then overrides with environment variables.
// Values will fall back to the provided defaults.
func LoadConfig() {
	// Tell Viper which file to read.
	viper.SetConfigFile("config.yml")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".") // look for the config file in the current directory

	// Attempt to read the configuration file; log if it doesn't exist.
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("No configuration file found or error reading file: %v", err)
	} else {
		log.Printf("Using config file: %s", viper.ConfigFileUsed())
	}

	// Allow environment variables to override configuration file values.
	viper.AutomaticEnv()

	// Set default values for AppConfig.
	viper.SetDefault("APP_PORT", "8080")
	viper.SetDefault("APP_LOG_LEVEL", "INFO")

	// Set default values for MySQLConfig.
	viper.SetDefault("MYSQL_HOST", "localhost")
	viper.SetDefault("MYSQL_PORT", 3306)
	viper.SetDefault("MYSQL_USER", "root")
	viper.SetDefault("MYSQL_PASSWORD", "")
	viper.SetDefault("MYSQL_DBNAME", "myapp_db")

	// Set default values for RedisConfig.
	viper.SetDefault("REDIS_HOST", "localhost")
	viper.SetDefault("REDIS_PORT", 6379)
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)

	// Initialize the global App config.
	App = AppConfig{
		Port:     viper.GetString("APP_PORT"),
		LogLevel: viper.GetString("APP_LOG_LEVEL"),
	}

	// Initialize the global MySQL config.
	MySQL = MySQLConfig{
		Host:     viper.GetString("MYSQL_HOST"),
		Port:     viper.GetInt("MYSQL_PORT"),
		User:     viper.GetString("MYSQL_USER"),
		Password: viper.GetString("MYSQL_PASSWORD"),
		DBName:   viper.GetString("MYSQL_DBNAME"),
	}

	// Initialize the global Redis config.
	Redis = RedisConfig{
		Host:     viper.GetString("REDIS_HOST"),
		Port:     viper.GetInt("REDIS_PORT"),
		Password: viper.GetString("REDIS_PASSWORD"),
		DB:       viper.GetInt("REDIS_DB"),
	}

	log.Printf("App Config: %+v", App)
	log.Printf("MySQL Config: %+v", MySQL)
	log.Printf("Redis Config: %+v", Redis)
}
