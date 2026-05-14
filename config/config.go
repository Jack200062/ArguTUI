package config

import (
	"errors"
	"fmt"
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
	// set defaults
	for _, inst := range c.Instances {
		if inst.LoginType == "" {
			inst.LoginType = LOGIN_TYPE_TOKEN
		}
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

		if inst.LoginType != LOGIN_TYPE_TOKEN && inst.LoginType != LOGIN_TYPE_CREDENTIALS && inst.LoginType != LOGIN_TYPE_SSO {
			return fmt.Errorf("loginType should be one of (%s, %s, %s)", LOGIN_TYPE_TOKEN, LOGIN_TYPE_CREDENTIALS, LOGIN_TYPE_SSO)
		}
	}
	return nil
}
