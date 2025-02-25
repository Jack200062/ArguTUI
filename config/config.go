package config

import (
	"errors"
	"strings"

	"github.com/Jack200062/ArguTUI/pkg/logging"
	"github.com/spf13/viper"
)

func Init(configPath string, logger *logging.Logger) (*Config, error) {
	v, err := loadConfig(configPath)
	if err != nil {
		return nil, logger.Errorf("failed to load config: %v", err)
	}
	c, err := parseConfig(v)
	if err != nil {
		return nil, logger.Errorf("failed to parse config: %v", err)
	}
	if err := validateConfig(c); err != nil {
		return nil, logger.Errorf("invalid config: %v", err)
	}
	return c, nil
}

func loadConfig(config string) (*viper.Viper, error) {
	v := viper.New()
	v.SetDefault("argocd.url", "localhost:8080")

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	if err := v.BindEnv("argocd.token"); err != nil {
		return nil, err
	}
	v.SetConfigFile(config)
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		var notFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &notFoundErr) {
			return nil, errors.New("config file not found")
		}
		return nil, err
	}
	return v, nil
}

func parseConfig(v *viper.Viper) (*Config, error) {
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func validateConfig(c *Config) error {
	if c.Argocd.Url == "" {
		return errors.New("argocd url is required")
	}
	if c.Argocd.Token == "" {
		return errors.New("argocd token is required")
	}
	return nil
}
