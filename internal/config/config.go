package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Auth          AuthConfig         `json:"auth" mapstructure:"auth"`
	Fabric        FabricConfig       `json:"fabric" mapstructure:"fabric"`
	Database      DatabaseConfig     `json:"database" mapstructure:"database"`
	UI            UIConfig           `json:"ui" mapstructure:"ui"`
	Notifications NotificationConfig `json:"notifications" mapstructure:"notifications"`
	Polling       PollingConfig      `json:"polling" mapstructure:"polling"`
	App           AppConfig          `json:"app" mapstructure:"app"`
}

// AuthConfig holds authentication-related configuration
type AuthConfig struct {
	ClientID    string `json:"clientId" mapstructure:"client_id"`
	TenantID    string `json:"tenantId" mapstructure:"tenant_id"`
	RedirectURI string `json:"redirectUri" mapstructure:"redirect_uri"`
}

// FabricConfig holds Fabric API-related configuration
type FabricConfig struct {
	WorkspaceIDs []string `json:"workspaceIds" mapstructure:"workspace_ids"`
	BaseURL      string   `json:"baseUrl" mapstructure:"base_url"`
}

// DatabaseConfig holds database-related configuration
type DatabaseConfig struct {
	Path          string `json:"path" mapstructure:"path"`
	EncryptionKey string `json:"encryptionKey" mapstructure:"encryption_key"`
	RetentionDays int    `json:"retentionDays" mapstructure:"retention_days"`
}

// UIConfig holds UI-related configuration
type UIConfig struct {
	Theme           string        `json:"theme" mapstructure:"theme"`
	PrimaryColor    string        `json:"primaryColor" mapstructure:"primary_color"`
	DefaultView     string        `json:"defaultView" mapstructure:"default_view"`
	RefreshInterval time.Duration `json:"refreshInterval" mapstructure:"refresh_interval"`
}

// NotificationConfig holds notification-related configuration
type NotificationConfig struct {
	Enabled              bool          `json:"enabled" mapstructure:"enabled"`
	OnFailure            bool          `json:"onFailure" mapstructure:"on_failure"`
	OnLongRunning        bool          `json:"onLongRunning" mapstructure:"on_long_running"`
	SoundEnabled         bool          `json:"soundEnabled" mapstructure:"sound_enabled"`
	LongRunningThreshold time.Duration `json:"longRunningThreshold" mapstructure:"long_running_threshold"`
}

// PollingConfig holds polling-related configuration
type PollingConfig struct {
	Interval time.Duration `json:"interval" mapstructure:"interval"`
	Enabled  bool          `json:"enabled" mapstructure:"enabled"`
}

// AppConfig holds general application configuration
type AppConfig struct {
	Debug    bool   `json:"debug" mapstructure:"debug"`
	LogLevel string `json:"logLevel" mapstructure:"log_level"`
	Name     string `json:"name" mapstructure:"name"`
	Version  string `json:"version" mapstructure:"version"`
}

// Load loads configuration from environment variables and config files
func Load() (*Config, error) {
	// Set defaults
	// Use a redirect URI that's commonly registered with public clients
	// The Azure CLI client accepts http://localhost:8400
	viper.SetDefault("auth.redirect_uri", "http://localhost:8400")
	viper.SetDefault("fabric.base_url", "https://api.fabric.microsoft.com/v1")
	viper.SetDefault("database.path", "data/fabric-monitor.db")
	viper.SetDefault("database.retention_days", 90)
	viper.SetDefault("ui.theme", "dark")
	viper.SetDefault("ui.primary_color", "#00BCF2")
	viper.SetDefault("ui.default_view", "dashboard")
	viper.SetDefault("ui.refresh_interval", "30s")
	viper.SetDefault("notifications.enabled", true)
	viper.SetDefault("notifications.on_failure", true)
	viper.SetDefault("notifications.on_long_running", false)
	viper.SetDefault("notifications.sound_enabled", true)
	viper.SetDefault("notifications.long_running_threshold", "30m")
	viper.SetDefault("polling.interval", "2m")
	viper.SetDefault("polling.enabled", true)
	viper.SetDefault("app.debug", false)
	viper.SetDefault("app.log_level", "info")
	viper.SetDefault("app.name", "Better Fabric Monitor")
	viper.SetDefault("app.version", "0.2.0")

	// Environment variable bindings
	viper.SetEnvPrefix("FABRIC_MONITOR")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Load from .env file if it exists
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		// .env file is optional, so ignore error if file not found
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Also try to load from config.yaml in app data directory
	configDir, err := getConfigDir()
	if err == nil {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(configDir)

		// This will override .env values if config.yaml exists
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return nil, fmt.Errorf("error reading config.yaml: %w", err)
			}
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Parse workspace IDs from comma-separated string
	if workspaceIDsStr := viper.GetString("fabric.workspace_ids"); workspaceIDsStr != "" {
		config.Fabric.WorkspaceIDs = strings.Split(workspaceIDsStr, ",")
		for i, id := range config.Fabric.WorkspaceIDs {
			config.Fabric.WorkspaceIDs[i] = strings.TrimSpace(id)
		}
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Save saves the configuration to the config file
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "config.yaml")

	viper.Set("auth", c.Auth)
	viper.Set("fabric", c.Fabric)
	viper.Set("database", c.Database)
	viper.Set("ui", c.UI)
	viper.Set("notifications", c.Notifications)
	viper.Set("polling", c.Polling)
	viper.Set("app", c.App)

	return viper.WriteConfigAs(configPath)
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// client_id is optional - app will use Microsoft PowerShell public client as fallback
	// tenant_id is optional - can be provided at login time
	if c.UI.PrimaryColor == "" {
		return fmt.Errorf("ui.primary_color is required")
	}
	return nil
}

// getConfigDir returns the application config directory
func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "fabric-monitor"), nil
}

// GetDataDir returns the application data directory
func GetDataDir() (string, error) {
	dataDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "fabric-monitor"), nil
}
