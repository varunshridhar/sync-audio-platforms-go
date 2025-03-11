package config

// AppConfig holds the application-level configuration.
type AppConfig struct {
	Port     string `json:"port" yaml:"port"`
	LogLevel string `json:"log_level" yaml:"log_level"`
}
