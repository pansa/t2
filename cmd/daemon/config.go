package main

import (
	"encoding/json"
	"os"
)

// Config struct is application config
type Config struct {
	Host      string           `json:"host"`
	Port      int              `json:"port"`
	Storage   *StorageConfig   `json:"storage"`
	RateLimit *RateLimitConfig `json:"rate_limit"`
	Redis     *RedisConfig     `json:"redis"`
}

// NewConfig func parse file and return Config pointer and error
func NewConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	jsonParser := json.NewDecoder(file)
	err = jsonParser.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
