package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the root configuration structure
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Directory DirectoryConfig `yaml:"directory"`
	Features  FeaturesConfig  `yaml:"features"`
	Logging   LoggingConfig   `yaml:"logging"`
}

type ServerConfig struct {
	Host string    `yaml:"host"`
	Port int       `yaml:"port"`
	TLS  TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

type DirectoryConfig struct {
	Path                 string `yaml:"path"`
	AllowSymlink         bool   `yaml:"allow_symlink"`
	AllowExternalSymlink bool   `yaml:"allow_external_symlink"`
	HideHidden           bool   `yaml:"hide_hidden"`
	RenderReadmeMd       bool   `yaml:"rander_readme_md"`
}

type FeaturesConfig struct {
	CORSEnabled bool           `yaml:"cors_enabled"`
	Compression []string       `yaml:"compression"`
	StatsFile   string         `yaml:"stats_file"`
	Auth        AuthConfig     `yaml:"auth"`
	Pages       map[int]string `yaml:"pages"`
	CacheRules  []CacheRule    `yaml:"cache_rules"`
}

type AuthConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type CacheRule struct {
	Pattern string `yaml:"pattern"`
	MaxAge  int    `yaml:"max_age"`
}

type LoggingConfig struct {
	Format    string `yaml:"format"`
	AccessLog string `yaml:"access_log"`
	ErrorLog  string `yaml:"error_log"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Write the default config file if it doesn't exist
			defaultConfig := CreateDefaultConfig()
			out, _ := yaml.Marshal(defaultConfig)
			if writeErr := os.WriteFile(path, out, 0644); writeErr != nil {
				return nil, fmt.Errorf("failed to create default config file: %w", writeErr)
			}
			fmt.Printf("Created default config file at %s\n", path)
			return defaultConfig, nil
		}
		return nil, fmt.Errorf("failed to read config file '%s': %w", path, err)
	}

	var conf Config
	if err := yaml.Unmarshal(data, &conf); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	return &conf, nil
}

func CreateDefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS: TLSConfig{
				Enabled:  false,
				CertFile: "/path/to/cert.pem",
				KeyFile:  "/path/to/key.pem",
			},
		},
		Directory: DirectoryConfig{
			Path:                 "./public",
			AllowSymlink:         true,
			AllowExternalSymlink: false,
			HideHidden:           true,
			RenderReadmeMd:       false,
		},
		Features: FeaturesConfig{
			CORSEnabled: false,
			Compression: []string{"gzip", "deflate", "brotli", "zstd"},
			StatsFile:   "data/stats.json",
			Auth: AuthConfig{
				Enabled:  false,
				Username: "admin",
				Password: "password123",
			},
			Pages: map[int]string{
				404: "./public/404.html",
			},
			CacheRules: []CacheRule{
				{Pattern: "*.html", MaxAge: 0},
				{Pattern: "*.png", MaxAge: 86400},
				{Pattern: "*", MaxAge: 3600},
			},
		},
		Logging: LoggingConfig{
			Format:    "plain",
			AccessLog: "access.log",
			ErrorLog:  "error.log",
		},
	}
}
