package config

type Instance struct {
	Name               string    `mapstructure:"name"`
	Url                string    `mapstructure:"url"`
	Token              string    `mapstructure:"token"`
	LoginType          LoginType `mapstructure:"logintype"`
	InsecureSkipVerify bool      `mapstructure:"insecureskipverify"`
}

type LoginType string

const (
	LOGIN_TYPE_TOKEN       LoginType = "token"
	LOGIN_TYPE_CREDENTIALS LoginType = "credentials"
	LOGIN_TYPE_SSO         LoginType = "sso"
)

type Config struct {
	Instances []*Instance `mapstructure:"instances"`
}
