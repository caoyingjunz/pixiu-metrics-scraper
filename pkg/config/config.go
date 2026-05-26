package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database DatabaseConfig `yaml:"database"`
	Server   ServerConfig   `yaml:"server"`
}

type DatabaseConfig struct {
	Type    string       `yaml:"type"` // "sqlite3" or "mysql"
	SQLite3 SQLiteConfig `yaml:"sqlite3"`
	MySQL   MySQLConfig  `yaml:"mysql"`
}

type SQLiteConfig struct {
	DBFile string `yaml:"db-file"`
}

type MySQLConfig struct {
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Host         string `yaml:"host"`
	Port         string `yaml:"port"`
	Database     string `yaml:"database"`
	Charset      string `yaml:"charset"`
	MaxOpenConns int    `yaml:"max-open-conns"`
	MaxIdleConns int    `yaml:"max-idle-conns"`
}

// ServerConfig represents server configuration
type ServerConfig struct {
	Address string `yaml:"address"`
}

func DefaultConfig() *Config {
	return &Config{
		Database: DatabaseConfig{
			Type: "sqlite3",
			SQLite3: SQLiteConfig{
				DBFile: "/tmp/metrics.db",
			},
			MySQL: MySQLConfig{
				Username:     "root",
				Password:     "",
				Host:         "localhost",
				Port:         "3306",
				Database:     "metrics_db",
				Charset:      "utf8mb4",
				MaxOpenConns: 25,
				MaxIdleConns: 10,
			},
		},
		Server: ServerConfig{
			Address: ":8000",
		},
	}
}

func LoadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{}
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return config, nil
}

func (c *MySQLConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=%s&parseTime=True&loc=UTC",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
		c.Charset,
	)
}
