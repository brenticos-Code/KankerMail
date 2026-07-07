package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// IMAPConfig stores settings for retrieving emails
type IMAPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

// SMTPConfig stores settings for sending emails
type SMTPConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	SSL      bool   `json:"ssl"`
}

// Config holds the program configuration
type Config struct {
	ThemeName string     `json:"theme_name"`
	UseMock   bool       `json:"use_mock"`
	HTMLRich  bool       `json:"html_rich"`
	IMAP      IMAPConfig `json:"imap"`
	SMTP      SMTPConfig `json:"smtp"`
}

const configFilename = "kankermail.json"

// GetConfigPath returns the path to the configuration file
func GetConfigPath() string {
	// For simplicity and ease of access by the user, we first check the current directory.
	// If we want it in a standard place, we can use os.UserConfigDir(), but let's stick to the current dir
	// or look for it in the current dir first.
	return configFilename
}

// DefaultConfig returns the default configuration
func DefaultConfig() Config {
	return Config{
		ThemeName: "Neon Dark",
		UseMock:   true,
		HTMLRich:  true,
		IMAP: IMAPConfig{
			Host:     "imap.example.com",
			Port:     993,
			Username: "user@example.com",
			Password: "",
			SSL:      true,
		},
		SMTP: SMTPConfig{
			Host:     "smtp.example.com",
			Port:     465,
			Username: "user@example.com",
			Password: "",
			SSL:      true,
		},
	}
}

// LoadConfig loads the configuration from the disk
func LoadConfig() (Config, error) {
	path := GetConfigPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Save default config first
			cfg := DefaultConfig()
			_ = SaveConfig(cfg)
			return cfg, nil
		}
		return DefaultConfig(), err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return DefaultConfig(), err
	}

	return cfg, nil
}

// SaveConfig saves the configuration to the disk
func SaveConfig(cfg Config) error {
	path := GetConfigPath()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	// Ensure directory exists if path is subfolder (not the case here but good practice)
	dir := filepath.Dir(path)
	if dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	return os.WriteFile(path, data, 0644)
}
