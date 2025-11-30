package utils

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		DBName   string `yaml:"dbname"`
	} `yaml:"database"`

	RPC struct {
		URL             string `yaml:"url"`
		ContractAddress string `yaml:"contract_address"`
		PrivateKey      string `yaml:"private_key"`
		UsdtAddress     string `yaml:"usdt_address"`
		UsdtDecimals    int    `yaml:"usdt_decimals"`
	} `yaml:"rpc"`

	Detector struct {
		ThresholdPercent float64 `yaml:"threshold_percent"`
		BodyRatioMax     float64 `yaml:"body_ratio_max"`
	} `yaml:"detector"`

	Chainlink struct {
		BtcUsdFeed     string `yaml:"btc_usd_feed"`
		UpdateInterval int    `yaml:"update_interval"`
	} `yaml:"chainlink"`

	EventListener struct {
		Enabled      bool `yaml:"enabled"`
		PollInterval int  `yaml:"poll_interval"`
	} `yaml:"eventlistener"`

	Mode string `yaml:"mode"`
}

var AppConfig *Config

// LoadConfig loads configuration from YAML file
func LoadConfig(path string) (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load(".env")

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Expand environment variables in the YAML content
	expandedData := os.ExpandEnv(string(data))

	var config Config
	if err := yaml.Unmarshal([]byte(expandedData), &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	AppConfig = &config
	return &config, nil
}

// LogInfo prints info level log
func LogInfo(format string, args ...interface{}) {
	log.Printf("[INFO] "+format, args...)
}

// LogError prints error level log with clickable file:line in VSCode terminal
func LogError(format string, args ...interface{}) {
	_, file, line, ok := runtime.Caller(1)
	if !ok {
		log.Printf("[ERROR] "+format, args...)
		return
	}
	// Use basename for cleaner log
	shortFile := file
	if lastSlash := strings.LastIndex(file, "/"); lastSlash != -1 {
		shortFile = file[lastSlash+1:]
	} else if lastBackslash := strings.LastIndex(file, "\\"); lastBackslash != -1 {
		shortFile = file[lastBackslash+1:]
	}
	log.Printf("[ERROR] %s:%d "+format, append([]interface{}{shortFile, line}, args...)...)
}

// LogDebug prints debug level log
func LogDebug(format string, args ...interface{}) {
	log.Printf("[DEBUG] "+format, args...)
}
