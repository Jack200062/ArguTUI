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

func loadConfig(configPath string) (*viper.Viper, error) {
	v := viper.New()

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.SetConfigFile(configPath)
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
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
	if len(c.Instances) == 0 {
		return errors.New("no instances provided in config")
	}
	for _, inst := range c.Instances {
		if inst.Url == "" {
			return errors.New("instance url is required")
		}
		if inst.Token == "" {
			return errors.New("instance token is required")
		}
	}
	return nil
}
