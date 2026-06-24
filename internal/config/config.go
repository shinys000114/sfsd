package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ServerInstance represents a single server configuration
type ServerInstance struct {
	Server    ServerConfig    `yaml:"server"`
	Directory DirectoryConfig `yaml:"directory"`
	Features  FeaturesConfig  `yaml:"features"`
	Logging   LoggingConfig   `yaml:"logging"`
}

// Config represents the root configuration containing multiple servers
type Config struct {
	Servers map[string]ServerInstance `yaml:",inline"`
}

type ServerConfig struct {
	Host    string    `yaml:"host"`
	Port    int       `yaml:"port"`
	Domains []string  `yaml:"domains,omitempty"`
	TLS     TLSConfig `yaml:"tls"`
}

type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	HTTP3    bool   `yaml:"http3"`
	CertFile string `yaml:"cert_file,omitempty"`
	KeyFile  string `yaml:"key_file,omitempty"`
}

type DirectoryConfig struct {
	Path                 string     `yaml:"path"`
	AllowSymlink         bool       `yaml:"allow_symlink"`
	AllowExternalSymlink bool       `yaml:"allow_external_symlink"`
	HideHidden           bool       `yaml:"hide_hidden"`
	RenderReadmeMd       bool       `yaml:"render_readme_md"`
	Exclude              []string   `yaml:"exclude,omitempty"`
	Icons                IconConfig `yaml:"icons,omitempty"`
}

type IconConfig struct {
	Directory  string            `yaml:"directory,omitempty"`
	Default    string            `yaml:"default,omitempty"`
	Extensions map[string]string `yaml:"extensions,omitempty"`
	MIMETypes  map[string]string `yaml:"mime_types,omitempty"`
}

type FeaturesConfig struct {
	CORSEnabled bool           `yaml:"cors_enabled"`
	Compression []string       `yaml:"compression"`
	StatsFile   string         `yaml:"stats_file,omitempty"`
	Auth        AuthConfig     `yaml:"auth"`
	Pages       map[int]string `yaml:"pages,omitempty"`
	CacheRules  []CacheRule    `yaml:"cache_rules,omitempty"`
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
	AccessLog string `yaml:"access_log,omitempty"`
	ErrorLog  string `yaml:"error_log,omitempty"`
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
	if err := yaml.Unmarshal(data, &conf.Servers); err != nil {
		return nil, fmt.Errorf("failed to parse yaml: %w", err)
	}

	return &conf, nil
}

func CreateDefaultConfig() *Config {
	instance := ServerInstance{
		Server: ServerConfig{
			Host: "localhost",
			Port: 8080,
			TLS: TLSConfig{
				Enabled: false,
				HTTP3:   false,
			},
		},
		Directory: DirectoryConfig{
			Path:                 "./public",
			AllowSymlink:         true,
			AllowExternalSymlink: false,
			HideHidden:           true,
			RenderReadmeMd:       false,
			Exclude:              []string{".git/", "*.tmp"},
		},
		Features: FeaturesConfig{
			CORSEnabled: false,
			Compression: []string{"gzip", "deflate", "brotli", "zstd"},
			Auth: AuthConfig{
				Enabled:  false,
				Username: "admin",
				Password: "password123",
			},
		},
		Logging: LoggingConfig{
			Format: "plain",
		},
	}

	return &Config{
		Servers: map[string]ServerInstance{
			"default": instance,
		},
	}
}
